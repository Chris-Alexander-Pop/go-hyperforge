package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/cloud/controlplane"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// MemoryControlPlane is an in-memory implementation of the ControlPlane interface.
type MemoryControlPlane struct {
	hosts     map[string]cloud.Host
	instances map[string]controlplane.Instance
	mu        *concurrency.SmartRWMutex
}

// New creates a new MemoryControlPlane.
func New() *MemoryControlPlane {
	return &MemoryControlPlane{
		hosts:     make(map[string]cloud.Host),
		instances: make(map[string]controlplane.Instance),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-controlplane",
		}),
	}
}

func (c *MemoryControlPlane) RegisterHost(ctx context.Context, host cloud.Host) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.hosts[host.ID]; ok {
		return controlplane.ErrHostAlreadyRegistered
	}
	if host.Available.VCPUs == 0 && host.Available.MemoryMB == 0 && host.Available.DiskGB == 0 {
		host.Available = host.Capacity
	}

	c.hosts[host.ID] = host
	return nil
}

func (c *MemoryControlPlane) DeregisterHost(ctx context.Context, hostID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.hosts[hostID]; !ok {
		return controlplane.ErrHostNotFound
	}
	for _, inst := range c.instances {
		if inst.HostID == hostID {
			return controlplane.ErrHostHasInstances
		}
	}

	delete(c.hosts, hostID)
	return nil
}

func (c *MemoryControlPlane) UpdateHostStatus(ctx context.Context, hostID string, status cloud.HostStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	host, ok := c.hosts[hostID]
	if !ok {
		return controlplane.ErrHostNotFound
	}

	host.Status = status
	c.hosts[hostID] = host
	return nil
}

func (c *MemoryControlPlane) GetHost(ctx context.Context, hostID string) (*cloud.Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	host, ok := c.hosts[hostID]
	if !ok {
		return nil, controlplane.ErrHostNotFound
	}

	return &host, nil
}

func (c *MemoryControlPlane) ListHosts(ctx context.Context) ([]cloud.Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hosts := make([]cloud.Host, 0, len(c.hosts))
	for _, h := range c.hosts {
		hosts = append(hosts, h)
	}
	return hosts, nil
}

func (c *MemoryControlPlane) CreateInstance(ctx context.Context, req controlplane.CreateInstanceRequest) (*controlplane.Instance, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if req.Name == "" {
		return nil, pkgerrors.InvalidArgument("instance name is required", nil)
	}

	inst := controlplane.Instance{
		ID:        uuid.NewString(),
		Name:      req.Name,
		Status:    cloud.InstanceStatusPending,
		Resources: req.Resources,
		Image:     req.Image,
		Tags:      req.Tags,
		CreatedAt: time.Now(),
	}

	if req.HostID != "" {
		if err := c.bindLocked(&inst, req.HostID); err != nil {
			return nil, err
		}
		inst.Status = cloud.InstanceStatusProvisioning
	}

	c.instances[inst.ID] = inst
	out := inst
	return &out, nil
}

func (c *MemoryControlPlane) BindInstance(ctx context.Context, instanceID, hostID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	inst, ok := c.instances[instanceID]
	if !ok {
		return controlplane.ErrInstanceNotFound
	}
	if inst.HostID != "" {
		return controlplane.ErrInstanceAlreadyBound
	}
	if err := c.bindLocked(&inst, hostID); err != nil {
		return err
	}
	if inst.Status == cloud.InstanceStatusPending {
		inst.Status = cloud.InstanceStatusProvisioning
	}
	c.instances[instanceID] = inst
	return nil
}

func (c *MemoryControlPlane) UnbindInstance(ctx context.Context, instanceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	inst, ok := c.instances[instanceID]
	if !ok {
		return controlplane.ErrInstanceNotFound
	}
	if inst.HostID == "" {
		return controlplane.ErrInstanceNotBound
	}
	c.releaseLocked(inst)
	inst.HostID = ""
	c.instances[instanceID] = inst
	return nil
}

func (c *MemoryControlPlane) GetInstance(ctx context.Context, instanceID string) (*controlplane.Instance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	inst, ok := c.instances[instanceID]
	if !ok {
		return nil, controlplane.ErrInstanceNotFound
	}
	out := inst
	return &out, nil
}

func (c *MemoryControlPlane) ListInstances(ctx context.Context, opts controlplane.ListInstancesOptions) ([]controlplane.Instance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]controlplane.Instance, 0, len(c.instances))
	for _, inst := range c.instances {
		if opts.HostID != "" && inst.HostID != opts.HostID {
			continue
		}
		if opts.Status != "" && inst.Status != opts.Status {
			continue
		}
		out = append(out, inst)
	}
	return out, nil
}

func (c *MemoryControlPlane) DeleteInstance(ctx context.Context, instanceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	inst, ok := c.instances[instanceID]
	if !ok {
		return controlplane.ErrInstanceNotFound
	}
	if inst.HostID != "" {
		c.releaseLocked(inst)
	}
	delete(c.instances, instanceID)
	return nil
}

func (c *MemoryControlPlane) UpdateInstanceStatus(ctx context.Context, instanceID string, status cloud.InstanceStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	inst, ok := c.instances[instanceID]
	if !ok {
		return controlplane.ErrInstanceNotFound
	}
	inst.Status = status
	c.instances[instanceID] = inst
	return nil
}

func (c *MemoryControlPlane) bindLocked(inst *controlplane.Instance, hostID string) error {
	host, ok := c.hosts[hostID]
	if !ok {
		return controlplane.ErrHostNotFound
	}
	if host.Status != cloud.HostStatusReady && host.Status != cloud.HostStatusBusy {
		return controlplane.ErrHostNotReady
	}
	res := inst.Resources
	if host.Available.VCPUs < res.VCPUs || host.Available.MemoryMB < res.MemoryMB || host.Available.DiskGB < res.DiskGB {
		return controlplane.ErrHostCapacityExhausted
	}
	host.Available.VCPUs -= res.VCPUs
	host.Available.MemoryMB -= res.MemoryMB
	host.Available.DiskGB -= res.DiskGB
	host.Available.GPUs -= res.GPUs
	if host.Available.VCPUs == 0 || host.Available.MemoryMB == 0 {
		host.Status = cloud.HostStatusBusy
	}
	c.hosts[hostID] = host
	inst.HostID = hostID
	return nil
}

func (c *MemoryControlPlane) releaseLocked(inst controlplane.Instance) {
	host, ok := c.hosts[inst.HostID]
	if !ok {
		return
	}
	host.Available.VCPUs += inst.Resources.VCPUs
	host.Available.MemoryMB += inst.Resources.MemoryMB
	host.Available.DiskGB += inst.Resources.DiskGB
	host.Available.GPUs += inst.Resources.GPUs
	if host.Status == cloud.HostStatusBusy {
		host.Status = cloud.HostStatusReady
	}
	c.hosts[inst.HostID] = host
}

var _ controlplane.ControlPlane = (*MemoryControlPlane)(nil)
