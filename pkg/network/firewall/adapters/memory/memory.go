package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/firewall"
	"github.com/google/uuid"
)

// MemoryFirewallManager is an in-memory implementation of FirewallManager.
type MemoryFirewallManager struct {
	groups map[string]*firewall.SecurityGroup
	mu     *concurrency.SmartRWMutex
}

// New creates a new MemoryFirewallManager.
func New() *MemoryFirewallManager {
	return &MemoryFirewallManager{
		groups: make(map[string]*firewall.SecurityGroup),
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{
			Name: "memory-firewall",
		}),
	}
}

func (m *MemoryFirewallManager) CreateSecurityGroup(ctx context.Context, spec firewall.SecurityGroupSpec) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.NewString()
	group := &firewall.SecurityGroup{
		ID:       id,
		Name:     spec.Name,
		VPCID:    spec.VPCID,
		Inbound:  []firewall.Rule{},
		Outbound: []firewall.Rule{},
		Tags:     spec.Tags,
	}

	m.groups[id] = group
	return id, nil
}

func (m *MemoryFirewallManager) DeleteSecurityGroup(ctx context.Context, groupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[groupID]; !ok {
		return firewall.ErrSecurityGroupNotFound
	}

	delete(m.groups, groupID)
	return nil
}

func (m *MemoryFirewallManager) AddRule(ctx context.Context, groupID string, rule firewall.Rule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[groupID]
	if !ok {
		return firewall.ErrSecurityGroupNotFound
	}

	if rule.Direction == "inbound" {
		group.Inbound = append(group.Inbound, rule)
	} else {
		group.Outbound = append(group.Outbound, rule)
	}
	return nil
}

func (m *MemoryFirewallManager) RemoveRule(ctx context.Context, groupID string, ruleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[groupID]
	if !ok {
		return firewall.ErrSecurityGroupNotFound
	}

	// Helper to remove from slice
	remove := func(rules []firewall.Rule) ([]firewall.Rule, bool) {
		for i, r := range rules {
			if r.ID == ruleID {
				return append(rules[:i], rules[i+1:]...), true
			}
		}
		return rules, false
	}

	var found bool
	group.Inbound, found = remove(group.Inbound)
	if !found {
		group.Outbound, found = remove(group.Outbound)
	}

	if !found {
		return firewall.ErrRuleNotFound
	}

	return nil
}
