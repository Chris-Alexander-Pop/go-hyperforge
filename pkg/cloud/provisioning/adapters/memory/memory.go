package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/provisioning"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

type MemoryProvisioner struct {
	hosts map[string]cloud.HostStatus
	mu    *concurrency.SmartRWMutex
}

// New creates a new in-memory provisioner.
func New() provisioning.Provisioner {
	return &MemoryProvisioner{
		hosts: make(map[string]cloud.HostStatus),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-provisioner",
		}),
	}
}

// AddHost is a helper for testing to seed the provisioner
func (p *MemoryProvisioner) AddHost(hostID string, status cloud.HostStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hosts[hostID] = status
}

func (p *MemoryProvisioner) ProvisionHost(ctx context.Context, hostID string, imageURL string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// In a real system, we'd check if the host exists in inventory.
	// For memory adapter, we simulate "discovering" it or erroring if strict.
	// Let's assume we can only provision known hosts.
	if _, ok := p.hosts[hostID]; !ok {
		return provisioning.ErrHostNotFound
	}

	p.hosts[hostID] = cloud.HostStatusBusy

	// Simulate async work
	go func() {
		time.Sleep(50 * time.Millisecond)
		p.mu.Lock()
		defer p.mu.Unlock()
		if _, ok := p.hosts[hostID]; ok {
			p.hosts[hostID] = cloud.HostStatusReady
		}
	}()

	return nil
}

func (p *MemoryProvisioner) DeprovisionHost(ctx context.Context, hostID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.hosts[hostID]; !ok {
		return provisioning.ErrHostNotFound
	}

	p.hosts[hostID] = cloud.HostStatusOffline
	return nil
}

func (p *MemoryProvisioner) GetHostStatus(ctx context.Context, hostID string) (cloud.HostStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status, ok := p.hosts[hostID]
	if !ok {
		return cloud.HostStatusUnknown, provisioning.ErrHostNotFound
	}

	return status, nil
}

func (p *MemoryProvisioner) PowerCycle(ctx context.Context, hostID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.hosts[hostID]; !ok {
		return provisioning.ErrHostNotFound
	}

	// Just a status check logic here for simulation
	return nil
}
