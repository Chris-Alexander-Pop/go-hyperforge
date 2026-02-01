package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cloud"
	"github.com/chris-alexander-pop/system-design-library/pkg/cloud/hypervisor"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/google/uuid"
)

// MemoryHypervisor is an in-memory implementation of the Hypervisor interface.
type MemoryHypervisor struct {
	vms       map[string]hypervisor.VM
	nameIndex map[string]string
	mu        *concurrency.SmartRWMutex
}

// New creates a new MemoryHypervisor.
func New() *MemoryHypervisor {
	return &MemoryHypervisor{
		vms:       make(map[string]hypervisor.VM),
		nameIndex: make(map[string]string),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-hypervisor",
		}),
	}
}

func (h *MemoryHypervisor) CreateVM(ctx context.Context, spec hypervisor.VMSpec) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for duplicate name
	if _, ok := h.nameIndex[spec.Name]; ok {
		return "", hypervisor.ErrVMAlreadyExists
	}

	id := uuid.NewString()
	vm := hypervisor.VM{
		ID:        id,
		Name:      spec.Name,
		Status:    cloud.InstanceStatusProvisioning, // Initial status
		Spec:      spec,
		CreatedAt: time.Now(),
		IPAddress: "192.168.1.100", // Fake IP
	}

	h.vms[id] = vm
	h.nameIndex[spec.Name] = id

	// Simulate provisioning delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.mu.Lock()
		defer h.mu.Unlock()
		if v, ok := h.vms[id]; ok {
			v.Status = cloud.InstanceStatusRunning
			h.vms[id] = v
		}
	}()

	return id, nil
}

func (h *MemoryHypervisor) StartVM(ctx context.Context, vmID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	vm, ok := h.vms[vmID]
	if !ok {
		return hypervisor.ErrVMNotFound
	}

	if vm.Status == cloud.InstanceStatusRunning {
		return nil
	}

	vm.Status = cloud.InstanceStatusRunning
	h.vms[vmID] = vm
	return nil
}

func (h *MemoryHypervisor) StopVM(ctx context.Context, vmID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	vm, ok := h.vms[vmID]
	if !ok {
		return hypervisor.ErrVMNotFound
	}

	if vm.Status == cloud.InstanceStatusStopped {
		return nil
	}

	vm.Status = cloud.InstanceStatusStopped
	h.vms[vmID] = vm
	return nil
}

func (h *MemoryHypervisor) DeleteVM(ctx context.Context, vmID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	vm, ok := h.vms[vmID]
	if !ok {
		return hypervisor.ErrVMNotFound
	}

	delete(h.vms, vmID)
	delete(h.nameIndex, vm.Name)
	return nil
}

func (h *MemoryHypervisor) GetVMStatus(ctx context.Context, vmID string) (cloud.InstanceStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	vm, ok := h.vms[vmID]
	if !ok {
		return cloud.InstanceStatusUnknown, hypervisor.ErrVMNotFound
	}

	return vm.Status, nil
}

func (h *MemoryHypervisor) ListVMs(ctx context.Context) ([]hypervisor.VM, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	vms := make([]hypervisor.VM, 0, len(h.vms))
	for _, vm := range h.vms {
		vms = append(vms, vm)
	}
	return vms, nil
}
