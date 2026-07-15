// Package memory provides an in-memory DeviceRegistry for tests and local use.
package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot/device/registry"
	"github.com/google/uuid"
)

// Registry is an in-memory DeviceRegistry.
type Registry struct {
	mu      *concurrency.SmartRWMutex
	devices map[string]*registry.Device
	closed  bool
}

var _ registry.DeviceRegistry = (*Registry)(nil)

// New creates an empty in-memory device registry.
func New() *Registry {
	return &Registry{
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "iot-device-registry"}),
		devices: make(map[string]*registry.Device),
	}
}

func (r *Registry) Register(ctx context.Context, opts registry.RegisterOptions) (*registry.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if opts.Name == "" && opts.ID == "" {
		return nil, registry.ErrInvalidDevice
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, registry.ErrInvalidDevice
	}
	id := opts.ID
	if id == "" {
		id = uuid.NewString()
	}
	if _, exists := r.devices[id]; exists {
		return nil, registry.ErrDeviceAlreadyExists
	}
	status := opts.Status
	if status == "" {
		status = registry.StatusProvisioned
	}
	now := time.Now()
	attrs := opts.Attributes
	if attrs == nil {
		attrs = map[string]string{}
	}
	d := &registry.Device{
		ID:            id,
		Name:          opts.Name,
		ThingType:     opts.ThingType,
		Status:        status,
		Attributes:    attrs,
		CertificateID: opts.CertificateID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	r.devices[id] = d
	return copyDevice(d), nil
}

func (r *Registry) Get(ctx context.Context, deviceID string) (*registry.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.devices[deviceID]
	if !ok {
		return nil, registry.ErrDeviceNotFound
	}
	return copyDevice(d), nil
}

func (r *Registry) Update(ctx context.Context, deviceID string, opts registry.UpdateOptions) (*registry.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[deviceID]
	if !ok {
		return nil, registry.ErrDeviceNotFound
	}
	if opts.Name != nil {
		d.Name = *opts.Name
	}
	if opts.ThingType != nil {
		d.ThingType = *opts.ThingType
	}
	if opts.Status != nil {
		d.Status = *opts.Status
	}
	if opts.CertificateID != nil {
		d.CertificateID = *opts.CertificateID
	}
	if opts.Attributes != nil {
		d.Attributes = opts.Attributes
	}
	d.UpdatedAt = time.Now()
	return copyDevice(d), nil
}

func (r *Registry) Deregister(ctx context.Context, deviceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.devices[deviceID]; !ok {
		return registry.ErrDeviceNotFound
	}
	delete(r.devices, deviceID)
	return nil
}

func (r *Registry) List(ctx context.Context, opts registry.ListOptions) ([]*registry.Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*registry.Device, 0, len(r.devices))
	for _, d := range r.devices {
		if opts.ThingType != "" && d.ThingType != opts.ThingType {
			continue
		}
		if opts.Status != "" && d.Status != opts.Status {
			continue
		}
		out = append(out, copyDevice(d))
	}
	if opts.Limit > 0 && len(out) > opts.Limit {
		out = out[:opts.Limit]
	}
	return out, nil
}

func (r *Registry) Touch(ctx context.Context, deviceID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.devices[deviceID]
	if !ok {
		return registry.ErrDeviceNotFound
	}
	now := time.Now()
	d.LastSeenAt = now
	d.UpdatedAt = now
	return nil
}

func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func copyDevice(d *registry.Device) *registry.Device {
	if d == nil {
		return nil
	}
	cp := *d
	if d.Attributes != nil {
		cp.Attributes = make(map[string]string, len(d.Attributes))
		for k, v := range d.Attributes {
			cp.Attributes[k] = v
		}
	}
	return &cp
}
