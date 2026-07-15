package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/dhcp"
	"github.com/google/uuid"
)

// MemoryIPAM is an in-memory implementation of IPAM.
type MemoryIPAM struct {
	allocations map[string]*dhcp.IPAllocation
	// Map poolID -> []Allocations for simpler lookup (not fully implemented in this stub)
	mu *concurrency.SmartRWMutex
}

// New creates a new MemoryIPAM.
func New() *MemoryIPAM {
	return &MemoryIPAM{
		allocations: make(map[string]*dhcp.IPAllocation),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-ipam",
		}),
	}
}

func (m *MemoryIPAM) AllocateIP(ctx context.Context, poolID string) (*dhcp.IPAllocation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Dumb allocator: just picks a random fake IP
	// Real implementation would look at available ranges in the pool
	id := uuid.NewString()
	ip := fmt.Sprintf("192.168.1.%d", len(m.allocations)+1)

	alloc := &dhcp.IPAllocation{
		ID:          id,
		PoolID:      poolID,
		IP:          ip,
		AllocatedAt: time.Now(),
	}

	m.allocations[id] = alloc
	return alloc, nil
}

func (m *MemoryIPAM) ReserveIP(ctx context.Context, poolID string, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already allocated
	for _, a := range m.allocations {
		if a.IP == ip && a.PoolID == poolID {
			return dhcp.ErrIPAlreadyAllocated
		}
	}

	id := uuid.NewString()
	alloc := &dhcp.IPAllocation{
		ID:          id,
		PoolID:      poolID,
		IP:          ip,
		AllocatedAt: time.Now(),
	}

	m.allocations[id] = alloc
	return nil
}

func (m *MemoryIPAM) ReleaseIP(ctx context.Context, allocationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.allocations[allocationID]; !ok {
		// Idempotent or error? Let's say idempotent success if not found usually,
		// but matching interface spec we might return error if stricter.
		return nil
	}

	delete(m.allocations, allocationID)
	return nil
}
