package consul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/servicemesh/discovery"
	"github.com/google/uuid"
)

// Config configures the Consul HTTP discovery adapter.
type Config struct {
	// Address is the Consul HTTP base URL (e.g. http://127.0.0.1:8500).
	Address string `env:"CONSUL_ADDRESS" env-default:"http://127.0.0.1:8500"`

	// Token is an optional ACL token (X-Consul-Token).
	Token string `env:"CONSUL_TOKEN"`

	// Datacenter selects a Consul datacenter (?dc=).
	Datacenter string `env:"CONSUL_DATACENTER"`

	// Namespace is an optional Consul Enterprise namespace.
	Namespace string `env:"CONSUL_NAMESPACE"`

	// HTTPClient overrides the default client (15s timeout).
	// Use discovery.WithMTLS to attach optional client certificates.
	HTTPClient *http.Client

	// WatchWait is the max blocking-query wait (default 5s; Consul accepts up to ~10m).
	WatchWait time.Duration
}

// Registry talks to Consul over HTTP.
type Registry struct {
	base   string
	token  string
	dc     string
	ns     string
	client *http.Client
	wait   time.Duration

	mu     sync.Mutex
	closed bool
}

var _ discovery.ServiceRegistry = (*Registry)(nil)

// New creates a Consul HTTP service registry.
func New(cfg Config) (*Registry, error) {
	addr := strings.TrimRight(cfg.Address, "/")
	if addr == "" {
		return nil, discovery.ErrInvalidService
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	wait := cfg.WatchWait
	if wait <= 0 {
		wait = 5 * time.Second
	}
	return &Registry{
		base:   addr,
		token:  cfg.Token,
		dc:     cfg.Datacenter,
		ns:     cfg.Namespace,
		client: client,
		wait:   wait,
	}, nil
}

type consulServiceDef struct {
	ID      string            `json:"ID"`
	Name    string            `json:"Name"`
	Address string            `json:"Address"`
	Port    int               `json:"Port"`
	Tags    []string          `json:"Tags,omitempty"`
	Meta    map[string]string `json:"Meta,omitempty"`
	Weights *consulWeights    `json:"Weights,omitempty"`
	Check   *consulCheck      `json:"Check,omitempty"`
}

type consulWeights struct {
	Passing int `json:"Passing"`
	Warning int `json:"Warning"`
}

type consulCheck struct {
	TTL                            string `json:"TTL,omitempty"`
	DeregisterCriticalServiceAfter string `json:"DeregisterCriticalServiceAfter,omitempty"`
}

type healthEntry struct {
	Service consulCatalogService `json:"Service"`
	Checks  []consulCheckStatus  `json:"Checks"`
}

type consulCatalogService struct {
	ID      string            `json:"ID"`
	Service string            `json:"Service"`
	Address string            `json:"Address"`
	Port    int               `json:"Port"`
	Tags    []string          `json:"Tags"`
	Meta    map[string]string `json:"Meta"`
	Weights *consulWeights    `json:"Weights"`
}

type consulCheckStatus struct {
	Status string `json:"Status"`
}

type agentService struct {
	ID      string            `json:"ID"`
	Service string            `json:"Service"`
	Address string            `json:"Address"`
	Port    int               `json:"Port"`
	Tags    []string          `json:"Tags"`
	Meta    map[string]string `json:"Meta"`
	Weights *consulWeights    `json:"Weights"`
}

func (r *Registry) Register(ctx context.Context, opts discovery.RegisterOptions) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if opts.Name == "" {
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

	def := consulServiceDef{
		ID:      id,
		Name:    opts.Name,
		Address: opts.Address,
		Port:    opts.Port,
		Tags:    opts.Tags,
		Meta:    opts.Metadata,
		Weights: &consulWeights{Passing: weight, Warning: 1},
	}
	if opts.TTL > 0 {
		def.Check = &consulCheck{TTL: opts.TTL.String()}
	} else if opts.HealthCheck != nil && opts.HealthCheck.Type == "ttl" && opts.HealthCheck.Interval > 0 {
		def.Check = &consulCheck{TTL: opts.HealthCheck.Interval.String()}
	}
	if opts.HealthCheck != nil && opts.HealthCheck.DeregisterCriticalServiceAfter > 0 {
		if def.Check == nil {
			def.Check = &consulCheck{}
		}
		def.Check.DeregisterCriticalServiceAfter = opts.HealthCheck.DeregisterCriticalServiceAfter.String()
	}

	body, err := json.Marshal(def)
	if err != nil {
		return nil, errors.InvalidArgument("failed to encode register payload", err)
	}
	resp, err := r.do(ctx, http.MethodPut, "/v1/agent/service/register", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, mapHTTPError(resp, "service registration failed")
	}

	now := time.Now()
	return &discovery.Service{
		ID:            id,
		Name:          opts.Name,
		Address:       opts.Address,
		Port:          opts.Port,
		Tags:          opts.Tags,
		Metadata:      opts.Metadata,
		Health:        discovery.HealthStatusPassing,
		Namespace:     r.ns,
		Weight:        weight,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}, nil
}

func (r *Registry) Deregister(ctx context.Context, serviceID string) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	path := "/v1/agent/service/deregister/" + url.PathEscape(serviceID)
	resp, err := r.do(ctx, http.MethodPut, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return discovery.ErrServiceNotFound
	}
	if resp.StatusCode >= 300 {
		return mapHTTPError(resp, "service deregistration failed")
	}
	return nil
}

func (r *Registry) Lookup(ctx context.Context, serviceName string, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceName == "" {
		return nil, discovery.ErrInvalidService
	}
	q := url.Values{}
	if opts.HealthyOnly {
		q.Set("passing", "true")
	}
	if opts.Tag != "" {
		q.Set("tag", opts.Tag)
	}
	path := "/v1/health/service/" + url.PathEscape(serviceName)
	resp, err := r.do(ctx, http.MethodGet, path, nil, q)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, mapHTTPError(resp, "service lookup failed")
	}
	var entries []healthEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&entries); err != nil {
		return nil, errors.Internal("failed to decode consul health response", err)
	}
	out := make([]*discovery.Service, 0, len(entries))
	for _, e := range entries {
		svc := fromCatalog(e.Service, r.ns, healthFromChecks(e.Checks))
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}
		out = append(out, svc)
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

func (r *Registry) Get(ctx context.Context, serviceID string) (*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	if serviceID == "" {
		return nil, discovery.ErrInvalidService
	}
	path := "/v1/agent/service/" + url.PathEscape(serviceID)
	resp, err := r.do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, discovery.ErrServiceNotFound
	}
	if resp.StatusCode >= 300 {
		return nil, mapHTTPError(resp, "get service failed")
	}
	var as agentService
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&as); err != nil {
		return nil, errors.Internal("failed to decode consul agent service", err)
	}
	return fromAgent(as, r.ns), nil
}

func (r *Registry) List(ctx context.Context, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	if err := r.errIfClosed(); err != nil {
		return nil, err
	}
	resp, err := r.do(ctx, http.MethodGet, "/v1/agent/services", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, mapHTTPError(resp, "list services failed")
	}
	var m map[string]agentService
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&m); err != nil {
		return nil, errors.Internal("failed to decode consul agent services", err)
	}
	out := make([]*discovery.Service, 0, len(m))
	for _, as := range m {
		svc := fromAgent(as, r.ns)
		if opts.Tag != "" && !containsTag(svc.Tags, opts.Tag) {
			continue
		}
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}
		out = append(out, svc)
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

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
	var index uint64
	for {
		if ctx.Err() != nil {
			return
		}
		r.mu.Lock()
		closed := r.closed
		r.mu.Unlock()
		if closed {
			return
		}

		q := url.Values{}
		q.Set("passing", "true")
		if index > 0 {
			q.Set("index", strconv.FormatUint(index, 10))
			q.Set("wait", formatConsulWait(r.wait))
		}
		path := "/v1/health/service/" + url.PathEscape(serviceName)
		resp, err := r.do(ctx, http.MethodGet, path, nil, q)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		if v := resp.Header.Get("X-Consul-Index"); v != "" {
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				index = n
			}
		}
		var entries []healthEntry
		if err := json.Unmarshal(body, &entries); err != nil {
			continue
		}
		services := make([]*discovery.Service, 0, len(entries))
		for _, e := range entries {
			services = append(services, fromCatalog(e.Service, r.ns, healthFromChecks(e.Checks)))
		}
		select {
		case ch <- services:
		case <-ctx.Done():
			return
		}
	}
}

func (r *Registry) Heartbeat(ctx context.Context, serviceID string) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	path := "/v1/agent/check/pass/service:" + url.PathEscape(serviceID)
	resp, err := r.do(ctx, http.MethodPut, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return discovery.ErrServiceNotFound
	}
	if resp.StatusCode >= 300 {
		return mapHTTPError(resp, "heartbeat failed")
	}
	return nil
}

func (r *Registry) UpdateHealth(ctx context.Context, serviceID string, status discovery.HealthStatus) error {
	if err := r.errIfClosed(); err != nil {
		return err
	}
	if serviceID == "" {
		return discovery.ErrInvalidService
	}
	var path string
	switch status {
	case discovery.HealthStatusPassing:
		path = "/v1/agent/check/pass/service:" + url.PathEscape(serviceID)
	case discovery.HealthStatusWarning:
		path = "/v1/agent/check/warn/service:" + url.PathEscape(serviceID)
	case discovery.HealthStatusCritical:
		path = "/v1/agent/check/fail/service:" + url.PathEscape(serviceID)
	default:
		return errors.InvalidArgument("unsupported health status", nil)
	}
	resp, err := r.do(ctx, http.MethodPut, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return discovery.ErrServiceNotFound
	}
	if resp.StatusCode >= 300 {
		return mapHTTPError(resp, "update health failed")
	}
	return nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *Registry) errIfClosed() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return discovery.ErrWatchClosed
	}
	return nil
}

func (r *Registry) do(ctx context.Context, method, path string, body io.Reader, q url.Values) (*http.Response, error) {
	if q == nil {
		q = url.Values{}
	}
	if r.dc != "" {
		q.Set("dc", r.dc)
	}
	if r.ns != "" {
		q.Set("ns", r.ns)
	}
	u := r.base + path
	if enc := q.Encode(); enc != "" {
		u += "?" + enc
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, errors.InvalidArgument("failed to build consul request", err)
	}
	if r.token != "" {
		req.Header.Set("X-Consul-Token", r.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.Unavailable("consul unreachable", err)
	}
	return resp, nil
}

func mapHTTPError(resp *http.Response, msg string) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	detail := strings.TrimSpace(string(b))
	if detail != "" {
		msg = fmt.Sprintf("%s: %s", msg, detail)
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		return discovery.ErrServiceNotFound
	case http.StatusBadRequest:
		return errors.InvalidArgument(msg, nil)
	case http.StatusForbidden, http.StatusUnauthorized:
		return errors.Unauthorized(msg, nil)
	default:
		return errors.Internal(msg, nil)
	}
}

func fromAgent(as agentService, ns string) *discovery.Service {
	weight := 1
	if as.Weights != nil && as.Weights.Passing > 0 {
		weight = as.Weights.Passing
	}
	id := as.ID
	if id == "" {
		id = as.Service
	}
	return &discovery.Service{
		ID:        id,
		Name:      as.Service,
		Address:   as.Address,
		Port:      as.Port,
		Tags:      as.Tags,
		Metadata:  as.Meta,
		Health:    discovery.HealthStatusPassing,
		Namespace: ns,
		Weight:    weight,
	}
}

func fromCatalog(cs consulCatalogService, ns string, health discovery.HealthStatus) *discovery.Service {
	weight := 1
	if cs.Weights != nil && cs.Weights.Passing > 0 {
		weight = cs.Weights.Passing
	}
	id := cs.ID
	if id == "" {
		id = cs.Service
	}
	return &discovery.Service{
		ID:        id,
		Name:      cs.Service,
		Address:   cs.Address,
		Port:      cs.Port,
		Tags:      cs.Tags,
		Metadata:  cs.Meta,
		Health:    health,
		Namespace: ns,
		Weight:    weight,
	}
}

func healthFromChecks(checks []consulCheckStatus) discovery.HealthStatus {
	if len(checks) == 0 {
		return discovery.HealthStatusPassing
	}
	worst := discovery.HealthStatusPassing
	for _, c := range checks {
		switch strings.ToLower(c.Status) {
		case "critical":
			return discovery.HealthStatusCritical
		case "warning":
			worst = discovery.HealthStatusWarning
		case "passing":
		default:
			if worst == discovery.HealthStatusPassing {
				worst = discovery.HealthStatusUnknown
			}
		}
	}
	return worst
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func formatConsulWait(d time.Duration) string {
	sec := int(d.Seconds())
	if sec < 1 {
		sec = 1
	}
	return strconv.Itoa(sec) + "s"
}
