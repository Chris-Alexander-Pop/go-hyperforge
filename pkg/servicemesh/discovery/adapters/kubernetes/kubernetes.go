package kubernetes

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config configures the Kubernetes discovery adapter.
type Config struct {
	// Namespace scopes Endpoints/EndpointSlice operations (default "default").
	Namespace string `env:"K8S_NAMESPACE" env-default:"default"`

	// PreferEndpointSlice uses discovery.k8s.io/v1 EndpointSlice when true.
	// Default false uses corev1.Endpoints (simpler fake-clientset tests).
	PreferEndpointSlice bool

	// RestConfig builds a real client when Client is nil.
	RestConfig *rest.Config

	// Client overrides RestConfig (required for fake clientset tests).
	Client kubernetes.Interface
}

// Registry implements discovery.ServiceRegistry via Kubernetes Endpoints.
type Registry struct {
	client    kubernetes.Interface
	namespace string
	useSlice  bool

	mu     *concurrency.SmartMutex
	closed bool
}

var _ discovery.ServiceRegistry = (*Registry)(nil)

// New creates a Kubernetes discovery registry.
func New(cfg Config) (*Registry, error) {
	ns := cfg.Namespace
	if ns == "" {
		ns = "default"
	}
	client := cfg.Client
	if client == nil {
		if cfg.RestConfig == nil {
			return nil, errors.InvalidArgument("kubernetes client or rest config is required", nil)
		}
		c, err := kubernetes.NewForConfig(cfg.RestConfig)
		if err != nil {
			return nil, errors.Unavailable("failed to build kubernetes client", err)
		}
		client = c
	}
	return &Registry{
		client:    client,
		namespace: ns,
		useSlice:  cfg.PreferEndpointSlice,
		mu:        concurrency.NewSmartMutex(concurrency.MutexConfig{Name: "discovery-kubernetes"}),
	}, nil
}

func (r *Registry) errIfClosed() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return discovery.ErrWatchClosed
	}
	return nil
}

// Register upserts an Endpoints object named after the service, adding this instance
// as an address subset. Creates a minimal Service if missing (for Endpoints ownership).
func (r *Registry) Register(ctx context.Context, opts discovery.RegisterOptions) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if opts.Name == "" || opts.Address == "" || opts.Port <= 0 {
		return nil, discovery.ErrInvalidService
	}
	id := opts.ID
	if id == "" {
		id = uuid.NewString()
	}
	weight := opts.Weight
	if weight <= 0 {
		weight = 1
	}
	now := time.Now().UTC()

	if err := r.ensureService(ctx, opts.Name, opts.Port); err != nil {
		return nil, err
	}

	ep, err := r.client.CoreV1().Endpoints(r.namespace).Get(ctx, opts.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		ep = &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      opts.Name,
				Namespace: r.namespace,
				Labels:    map[string]string{"hyperforge.io/discovery": "true"},
			},
			Subsets: []corev1.EndpointSubset{},
		}
		ep, err = r.client.CoreV1().Endpoints(r.namespace).Create(ctx, ep, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Unavailable("failed to create endpoints", err)
		}
	} else if err != nil {
		return nil, errors.Unavailable("failed to get endpoints", err)
	}

	// Remove any prior address with same ID annotation key in targetRef name.
	ep.Subsets = removeAddress(ep.Subsets, id)
	ep.Subsets = append(ep.Subsets, corev1.EndpointSubset{
		Addresses: []corev1.EndpointAddress{{
			IP: opts.Address,
			TargetRef: &corev1.ObjectReference{
				Kind: "Pod",
				Name: id,
				UID:  "",
			},
		}},
		Ports: []corev1.EndpointPort{{
			Port:     int32(opts.Port),
			Protocol: corev1.ProtocolTCP,
			Name:     "http",
		}},
	})
	if ep.Annotations == nil {
		ep.Annotations = map[string]string{}
	}
	ep.Annotations["hyperforge.io/instance/"+id] = encodeMeta(opts)

	_, err = r.client.CoreV1().Endpoints(r.namespace).Update(ctx, ep, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Unavailable("failed to update endpoints", err)
	}

	return &discovery.Service{
		ID:            id,
		Name:          opts.Name,
		Address:       opts.Address,
		Port:          opts.Port,
		Tags:          opts.Tags,
		Metadata:      opts.Metadata,
		Health:        discovery.HealthStatusPassing,
		Namespace:     r.namespace,
		Weight:        weight,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}, nil
}

func (r *Registry) ensureService(ctx context.Context, name string, port int) error {
	_, err := r.client.CoreV1().Services(r.namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return errors.Unavailable("failed to get service", err)
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.namespace,
			Labels:    map[string]string{"hyperforge.io/discovery": "true"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name: "http",
				Port: int32(port),
			}},
			Selector: map[string]string{"app": name},
		},
	}
	_, err = r.client.CoreV1().Services(r.namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Unavailable("failed to create service", err)
	}
	return nil
}

func encodeMeta(opts discovery.RegisterOptions) string {
	// Compact annotation: weight|tag1,tag2
	w := opts.Weight
	if w <= 0 {
		w = 1
	}
	tags := ""
	for i, t := range opts.Tags {
		if i > 0 {
			tags += ","
		}
		tags += t
	}
	return fmt.Sprintf("%d|%s", w, tags)
}

func removeAddress(subsets []corev1.EndpointSubset, id string) []corev1.EndpointSubset {
	out := make([]corev1.EndpointSubset, 0, len(subsets))
	for _, sub := range subsets {
		addrs := make([]corev1.EndpointAddress, 0, len(sub.Addresses))
		for _, a := range sub.Addresses {
			if a.TargetRef != nil && a.TargetRef.Name == id {
				continue
			}
			addrs = append(addrs, a)
		}
		if len(addrs) == 0 {
			continue
		}
		sub.Addresses = addrs
		out = append(out, sub)
	}
	return out
}

// Deregister removes the instance address from Endpoints.
func (r *Registry) Deregister(ctx context.Context, serviceID string) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	list, err := r.client.CoreV1().Endpoints(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Unavailable("failed to list endpoints", err)
	}
	for i := range list.Items {
		ep := &list.Items[i]
		if !hasInstance(ep, serviceID) {
			continue
		}
		ep.Subsets = removeAddress(ep.Subsets, serviceID)
		if ep.Annotations != nil {
			delete(ep.Annotations, "hyperforge.io/instance/"+serviceID)
		}
		_, err = r.client.CoreV1().Endpoints(r.namespace).Update(ctx, ep, metav1.UpdateOptions{})
		if err != nil {
			return errors.Unavailable("failed to update endpoints", err)
		}
		return nil
	}
	return discovery.ErrServiceNotFound
}

func hasInstance(ep *corev1.Endpoints, serviceID string) bool {
	for _, sub := range ep.Subsets {
		for _, a := range sub.Addresses {
			if a.TargetRef != nil && a.TargetRef.Name == serviceID {
				return true
			}
		}
	}
	return false
}

// Lookup returns instances from Endpoints or EndpointSlices for serviceName.
func (r *Registry) Lookup(ctx context.Context, serviceName string, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceName == "" {
		return nil, discovery.ErrInvalidService
	}
	if r.useSlice {
		return r.lookupSlices(ctx, serviceName, opts)
	}
	ep, err := r.client.CoreV1().Endpoints(r.namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return []*discovery.Service{}, nil
	}
	if err != nil {
		return nil, errors.Unavailable("failed to get endpoints", err)
	}
	return filterServices(fromEndpoints(ep, r.namespace), opts), nil
}

func (r *Registry) lookupSlices(ctx context.Context, serviceName string, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	list, err := r.client.DiscoveryV1().EndpointSlices(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kubernetes.io/service-name=" + serviceName,
	})
	if err != nil {
		return nil, errors.Unavailable("failed to list endpoint slices", err)
	}
	svcs := make([]*discovery.Service, 0)
	for i := range list.Items {
		svcs = append(svcs, fromEndpointSlice(&list.Items[i], serviceName, r.namespace)...)
	}
	return filterServices(svcs, opts), nil
}

// Get finds an instance by ID across Endpoints in the namespace.
func (r *Registry) Get(ctx context.Context, serviceID string) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, discovery.ErrInvalidService
	}
	list, err := r.client.CoreV1().Endpoints(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Unavailable("failed to list endpoints", err)
	}
	for i := range list.Items {
		for _, svc := range fromEndpoints(&list.Items[i], r.namespace) {
			if svc.ID == serviceID {
				return svc, nil
			}
		}
	}
	return nil, discovery.ErrServiceNotFound
}

// List returns all instances from all Endpoints in the namespace.
func (r *Registry) List(ctx context.Context, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	list, err := r.client.CoreV1().Endpoints(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Unavailable("failed to list endpoints", err)
	}
	out := make([]*discovery.Service, 0)
	for i := range list.Items {
		out = append(out, fromEndpoints(&list.Items[i], r.namespace)...)
	}
	return filterServices(out, opts), nil
}

// Watch watches Endpoints (or EndpointSlices) for serviceName.
func (r *Registry) Watch(ctx context.Context, serviceName string) (<-chan []*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceName == "" {
		return nil, discovery.ErrInvalidService
	}
	ch := make(chan []*discovery.Service, 4)
	go r.watchLoop(ctx, serviceName, ch)
	return ch, nil
}

func (r *Registry) watchLoop(ctx context.Context, serviceName string, ch chan []*discovery.Service) {
	defer close(ch)
	emit := func() {
		svcs, err := r.Lookup(ctx, serviceName, discovery.QueryOptions{})
		if err != nil {
			return
		}
		select {
		case ch <- svcs:
		case <-ctx.Done():
		}
	}
	emit()

	var w watch.Interface
	var err error
	if r.useSlice {
		w, err = r.client.DiscoveryV1().EndpointSlices(r.namespace).Watch(ctx, metav1.ListOptions{
			LabelSelector: "kubernetes.io/service-name=" + serviceName,
		})
	} else {
		w, err = r.client.CoreV1().Endpoints(r.namespace).Watch(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + serviceName,
		})
	}
	if err != nil {
		// Fallback poll if Watch unsupported (some fakes).
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.mu.Lock()
				closed := r.closed
				r.mu.Unlock()
				if closed {
					return
				}
				emit()
			}
		}
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-w.ResultChan():
			if !ok {
				return
			}
			r.mu.Lock()
			closed := r.closed
			r.mu.Unlock()
			if closed {
				return
			}
			if ev.Type == watch.Error {
				continue
			}
			emit()
		}
	}
}

// Heartbeat is a no-op for Kubernetes Endpoints (kubelet owns readiness).
func (r *Registry) Heartbeat(ctx context.Context, serviceID string) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	_, err := r.Get(ctx, serviceID)
	return err
}

// UpdateHealth is a no-op stub (readiness is owned by the kubelet/probes).
func (r *Registry) UpdateHealth(ctx context.Context, serviceID string, status discovery.HealthStatus) error {
	_ = status
	return r.Heartbeat(ctx, serviceID)
}

// Close marks the registry closed.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func fromEndpoints(ep *corev1.Endpoints, ns string) []*discovery.Service {
	out := make([]*discovery.Service, 0)
	for _, sub := range ep.Subsets {
		port := 0
		if len(sub.Ports) > 0 {
			port = int(sub.Ports[0].Port)
		}
		for _, a := range sub.Addresses {
			id := a.IP + ":" + strconv.Itoa(port)
			if a.TargetRef != nil && a.TargetRef.Name != "" {
				id = a.TargetRef.Name
			}
			weight := 1
			var tags []string
			if ep.Annotations != nil {
				if raw, ok := ep.Annotations["hyperforge.io/instance/"+id]; ok {
					weight, tags = decodeMeta(raw)
				}
			}
			out = append(out, &discovery.Service{
				ID:        id,
				Name:      ep.Name,
				Address:   a.IP,
				Port:      port,
				Tags:      tags,
				Health:    discovery.HealthStatusPassing,
				Namespace: ns,
				Weight:    weight,
			})
		}
	}
	return out
}

func fromEndpointSlice(slice *discoveryv1.EndpointSlice, serviceName, ns string) []*discovery.Service {
	out := make([]*discovery.Service, 0)
	port := 0
	if len(slice.Ports) > 0 && slice.Ports[0].Port != nil {
		port = int(*slice.Ports[0].Port)
	}
	for _, ep := range slice.Endpoints {
		for _, addr := range ep.Addresses {
			id := addr + ":" + strconv.Itoa(port)
			if ep.TargetRef != nil && ep.TargetRef.Name != "" {
				id = ep.TargetRef.Name
			}
			out = append(out, &discovery.Service{
				ID:        id,
				Name:      serviceName,
				Address:   addr,
				Port:      port,
				Health:    discovery.HealthStatusPassing,
				Namespace: ns,
				Weight:    1,
			})
		}
	}
	return out
}

func decodeMeta(raw string) (int, []string) {
	weight := 1
	var tags []string
	parts := split2(raw, '|')
	if len(parts) >= 1 {
		if w, err := strconv.Atoi(parts[0]); err == nil && w > 0 {
			weight = w
		}
	}
	if len(parts) >= 2 && parts[1] != "" {
		for _, t := range splitComma(parts[1]) {
			if t != "" {
				tags = append(tags, t)
			}
		}
	}
	return weight, tags
}

func split2(s string, sep byte) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func splitComma(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func filterServices(svcs []*discovery.Service, opts discovery.QueryOptions) []*discovery.Service {
	out := make([]*discovery.Service, 0, len(svcs))
	for _, svc := range svcs {
		if opts.Tag != "" && !containsTag(svc.Tags, opts.Tag) {
			continue
		}
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}
		if opts.HealthyOnly && svc.Health != discovery.HealthStatusPassing {
			continue
		}
		out = append(out, svc)
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}
