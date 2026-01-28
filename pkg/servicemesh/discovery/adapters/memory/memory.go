package memory

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/servicemesh/discovery"
	"github.com/google/uuid"
)

// Registry implements an in-memory service registry for testing.
type Registry struct {
	mu       sync.RWMutex
	services map[string]*discovery.Service
	byName   map[string][]string // name -> []serviceID
	watchers map[string][]chan []*discovery.Service
	config   discovery.Config
	closed   bool
}

// New creates a new in-memory service registry.
func New() *Registry {
	return &Registry{
		services: make(map[string]*discovery.Service),
		byName:   make(map[string][]string),
		watchers: make(map[string][]chan []*discovery.Service),
		config:   discovery.Config{TTL: 30 * time.Second, Namespace: "default"},
	}
}

func (r *Registry) Register(ctx context.Context, opts discovery.RegisterOptions) (*discovery.Service, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if opts.Name == "" {
		return nil, errors.InvalidArgument("service name is required", nil)
	}

	id := opts.ID
	if id == "" {
		id = uuid.NewString()
	}

	weight := opts.Weight
	if weight <= 0 {
		weight = 1
	}

	now := time.Now()
	svc := &discovery.Service{
		ID:            id,
		Name:          opts.Name,
		Address:       opts.Address,
		Port:          opts.Port,
		Tags:          opts.Tags,
		Metadata:      opts.Metadata,
		Health:        discovery.HealthStatusPassing,
		Namespace:     r.config.Namespace,
		Weight:        weight,
		RegisteredAt:  now,
		LastHeartbeat: now,
	}

	r.services[id] = svc
	r.byName[opts.Name] = append(r.byName[opts.Name], id)

	// Notify watchers
	r.notifyWatchers(opts.Name)

	return svc, nil
}

func (r *Registry) Deregister(ctx context.Context, serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc, ok := r.services[serviceID]
	if !ok {
		return errors.NotFound("service not found", nil)
	}

	name := svc.Name
	delete(r.services, serviceID)

	// Remove from byName index
	ids := r.byName[name]
	for i, id := range ids {
		if id == serviceID {
			r.byName[name] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	// Notify watchers
	r.notifyWatchers(name)

	return nil
}

func (r *Registry) Lookup(ctx context.Context, serviceName string, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids, ok := r.byName[serviceName]
	if !ok {
		return []*discovery.Service{}, nil
	}

	var result []*discovery.Service
	for _, id := range ids {
		svc := r.services[id]
		if svc == nil {
			continue
		}

		// Apply filters
		if opts.HealthyOnly && svc.Health != discovery.HealthStatusPassing {
			continue
		}
		if opts.Tag != "" && !containsTag(svc.Tags, opts.Tag) {
			continue
		}
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}

		result = append(result, svc)
	}

	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (r *Registry) Get(ctx context.Context, serviceID string) (*discovery.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.services[serviceID]
	if !ok {
		return nil, errors.NotFound("service not found", nil)
	}

	return svc, nil
}

func (r *Registry) List(ctx context.Context, opts discovery.QueryOptions) ([]*discovery.Service, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*discovery.Service
	for _, svc := range r.services {
		if opts.HealthyOnly && svc.Health != discovery.HealthStatusPassing {
			continue
		}
		if opts.Tag != "" && !containsTag(svc.Tags, opts.Tag) {
			continue
		}
		if opts.Namespace != "" && svc.Namespace != opts.Namespace {
			continue
		}
		result = append(result, svc)
	}

	if opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

func (r *Registry) Watch(ctx context.Context, serviceName string) (<-chan []*discovery.Service, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, errors.Internal("registry closed", nil)
	}

	ch := make(chan []*discovery.Service, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], ch)

	// Send initial state
	go func() {
		defer func() {
			// Handle send on closed channel if Close() is called
			_ = recover()
		}()
		services, _ := r.Lookup(ctx, serviceName, discovery.QueryOptions{})
		select {
		case ch <- services:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

func (r *Registry) notifyWatchers(serviceName string) {
	watchers := r.watchers[serviceName]
	if len(watchers) == 0 {
		return
	}

	ids := r.byName[serviceName]
	var services []*discovery.Service
	for _, id := range ids {
		if svc := r.services[id]; svc != nil {
			services = append(services, svc)
		}
	}

	for _, ch := range watchers {
		select {
		case ch <- services:
		default:
			// Channel full, skip
		}
	}
}

func (r *Registry) Heartbeat(ctx context.Context, serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc, ok := r.services[serviceID]
	if !ok {
		return errors.NotFound("service not found", nil)
	}

	svc.LastHeartbeat = time.Now()
	svc.Health = discovery.HealthStatusPassing

	return nil
}

func (r *Registry) UpdateHealth(ctx context.Context, serviceID string, status discovery.HealthStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc, ok := r.services[serviceID]
	if !ok {
		return errors.NotFound("service not found", nil)
	}

	svc.Health = status
	r.notifyWatchers(svc.Name)

	return nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true

	// Close all watcher channels
	for _, watchers := range r.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}
	r.watchers = make(map[string][]chan []*discovery.Service)

	return nil
}
