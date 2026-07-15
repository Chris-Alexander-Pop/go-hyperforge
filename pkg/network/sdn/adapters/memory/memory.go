package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/sdn"
	"github.com/google/uuid"
)

// MemoryNetworkManager is an in-memory implementation of NetworkManager.
type MemoryNetworkManager struct {
	networks map[string]*sdn.Network
	mu       *concurrency.SmartRWMutex
}

// New creates a new MemoryNetworkManager.
func New() *MemoryNetworkManager {
	return &MemoryNetworkManager{
		networks: make(map[string]*sdn.Network),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-sdn",
		}),
	}
}

func (m *MemoryNetworkManager) CreateNetwork(ctx context.Context, spec sdn.NetworkSpec) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing network with same CIDR (simplified collision check)
	for _, n := range m.networks {
		if n.CIDR == spec.CIDR {
			return "", sdn.ErrNetworkAlreadyExists
		}
	}

	id := uuid.NewString()
	network := &sdn.Network{
		ID:      id,
		Name:    spec.Name,
		CIDR:    spec.CIDR,
		Subnets: []sdn.Subnet{},
		Tags:    spec.Tags,
	}

	m.networks[id] = network
	return id, nil
}

func (m *MemoryNetworkManager) DeleteNetwork(ctx context.Context, networkID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.networks[networkID]; !ok {
		return sdn.ErrNetworkNotFound
	}

	delete(m.networks, networkID)
	return nil
}

func (m *MemoryNetworkManager) CreateSubnet(ctx context.Context, networkID string, spec sdn.SubnetSpec) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	network, ok := m.networks[networkID]
	if !ok {
		return "", sdn.ErrNetworkNotFound
	}

	// Simplistic overlap check would go here

	id := uuid.NewString()
	subnet := sdn.Subnet{
		ID:        id,
		NetworkID: networkID,
		Name:      spec.Name,
		CIDR:      spec.CIDR,
		Zone:      spec.Zone,
		Tags:      spec.Tags,
	}

	network.Subnets = append(network.Subnets, subnet)
	return id, nil
}

func (m *MemoryNetworkManager) DeleteSubnet(ctx context.Context, subnetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// This is O(N) scan, acceptable for memory adapter
	for _, network := range m.networks {
		for i, subnet := range network.Subnets {
			if subnet.ID == subnetID {
				// Remove from slice
				network.Subnets = append(network.Subnets[:i], network.Subnets[i+1:]...)
				return nil
			}
		}
	}

	return sdn.ErrSubnetNotFound
}

func (m *MemoryNetworkManager) GetNetwork(ctx context.Context, networkID string) (*sdn.Network, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	network, ok := m.networks[networkID]
	if !ok {
		return nil, sdn.ErrNetworkNotFound
	}

	// Return copy to prevent race conditions
	n := *network
	return &n, nil
}
