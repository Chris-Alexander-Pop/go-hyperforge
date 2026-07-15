package memory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/leastconnections"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/random"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/roundrobin"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/weightedroundrobin"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer"
	"github.com/google/uuid"
)

// Manager implements an in-memory load balancer manager for testing and
// in-process selection. Target selection reuses pkg/algorithms/loadbalancing
// (round-robin, least-connections, weighted round-robin, random).
type Manager struct {
	mu            *concurrency.SmartRWMutex
	loadBalancers map[string]*loadbalancer.LoadBalancer
	listeners     map[string]*loadbalancer.Listener // listenerID -> listener
	targetPools   map[string]*loadbalancer.TargetPool
	rules         map[string][]*loadbalancer.Rule   // listenerID -> rules
	balancers     map[string]loadbalancing.Balancer // poolID -> algorithm balancer
	config        loadbalancer.Config
}

// New creates a new in-memory load balancer manager.
func New() *Manager {
	return &Manager{
		mu:            concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-loadbalancer"}),
		loadBalancers: make(map[string]*loadbalancer.LoadBalancer),
		listeners:     make(map[string]*loadbalancer.Listener),
		targetPools:   make(map[string]*loadbalancer.TargetPool),
		rules:         make(map[string][]*loadbalancer.Rule),
		balancers:     make(map[string]loadbalancing.Balancer),
		config:        loadbalancer.Config{},
	}
}

func newBalancer(algo loadbalancer.Algorithm) (loadbalancing.Balancer, error) {
	switch algo {
	case "", loadbalancer.AlgorithmRoundRobin:
		return roundrobin.New(), nil
	case loadbalancer.AlgorithmLeastConnections:
		return leastconnections.New(), nil
	case loadbalancer.AlgorithmWeightedRoundRobin:
		return weightedroundrobin.New(), nil
	case loadbalancer.AlgorithmRandom:
		return random.New(), nil
	default:
		return nil, loadbalancer.ErrUnsupportedAlgorithm
	}
}

func (m *Manager) CreateLoadBalancer(ctx context.Context, opts loadbalancer.CreateLoadBalancerOptions) (*loadbalancer.LoadBalancer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb := &loadbalancer.LoadBalancer{
		ID:        uuid.NewString(),
		Name:      opts.Name,
		DNSName:   fmt.Sprintf("%s.lb.local", opts.Name),
		Scheme:    opts.Scheme,
		Type:      opts.Type,
		State:     "active",
		Listeners: []*loadbalancer.Listener{},
		Tags:      opts.Tags,
		CreatedAt: time.Now(),
	}

	m.loadBalancers[lb.ID] = lb
	return lb, nil
}

func (m *Manager) GetLoadBalancer(ctx context.Context, id string) (*loadbalancer.LoadBalancer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lb, ok := m.loadBalancers[id]
	if !ok {
		return nil, loadbalancer.ErrLoadBalancerNotFound
	}

	return lb, nil
}

func (m *Manager) ListLoadBalancers(ctx context.Context) ([]*loadbalancer.LoadBalancer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lbs := make([]*loadbalancer.LoadBalancer, 0, len(m.loadBalancers))
	for _, lb := range m.loadBalancers {
		lbs = append(lbs, lb)
	}

	return lbs, nil
}

func (m *Manager) DeleteLoadBalancer(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, ok := m.loadBalancers[id]
	if !ok {
		return loadbalancer.ErrLoadBalancerNotFound
	}

	if len(lb.Listeners) > 0 {
		return loadbalancer.ErrLoadBalancerInUse
	}

	delete(m.loadBalancers, id)
	return nil
}

func (m *Manager) CreateListener(ctx context.Context, opts loadbalancer.CreateListenerOptions) (*loadbalancer.Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, ok := m.loadBalancers[opts.LoadBalancerID]
	if !ok {
		return nil, loadbalancer.ErrLoadBalancerNotFound
	}

	listener := &loadbalancer.Listener{
		ID:                uuid.NewString(),
		LoadBalancerID:    opts.LoadBalancerID,
		Protocol:          opts.Protocol,
		Port:              opts.Port,
		TargetPoolID:      opts.TargetPoolID,
		SSLCertificateARN: opts.SSLCertificateARN,
		Rules:             []*loadbalancer.Rule{},
		CreatedAt:         time.Now(),
	}

	lb.Listeners = append(lb.Listeners, listener)
	m.listeners[listener.ID] = listener

	return listener, nil
}

func (m *Manager) DeleteListener(ctx context.Context, loadBalancerID, listenerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, ok := m.loadBalancers[loadBalancerID]
	if !ok {
		return loadbalancer.ErrLoadBalancerNotFound
	}

	if _, ok := m.listeners[listenerID]; !ok {
		return loadbalancer.ErrListenerNotFound
	}

	// Remove from load balancer
	for i, l := range lb.Listeners {
		if l.ID == listenerID {
			lb.Listeners = append(lb.Listeners[:i], lb.Listeners[i+1:]...)
			break
		}
	}

	delete(m.listeners, listenerID)
	delete(m.rules, listenerID)

	return nil
}

func (m *Manager) CreateTargetPool(ctx context.Context, opts loadbalancer.CreateTargetPoolOptions) (*loadbalancer.TargetPool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	algo := opts.Algorithm
	if algo == "" {
		algo = loadbalancer.AlgorithmRoundRobin
	}

	bal, err := newBalancer(algo)
	if err != nil {
		return nil, err
	}

	pool := &loadbalancer.TargetPool{
		ID:          uuid.NewString(),
		Name:        opts.Name,
		Protocol:    opts.Protocol,
		Port:        opts.Port,
		Algorithm:   algo,
		HealthCheck: opts.HealthCheck,
		Targets:     []*loadbalancer.Target{},
		Tags:        opts.Tags,
		CreatedAt:   time.Now(),
	}

	m.targetPools[pool.ID] = pool
	m.balancers[pool.ID] = bal
	return pool, nil
}

func (m *Manager) GetTargetPool(ctx context.Context, id string) (*loadbalancer.TargetPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.targetPools[id]
	if !ok {
		return nil, loadbalancer.ErrTargetPoolNotFound
	}

	return pool, nil
}

func (m *Manager) DeleteTargetPool(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[id]
	if !ok {
		return loadbalancer.ErrTargetPoolNotFound
	}

	if len(pool.Targets) > 0 {
		return loadbalancer.ErrTargetPoolInUse
	}

	delete(m.targetPools, id)
	delete(m.balancers, id)
	return nil
}

func (m *Manager) AddTarget(ctx context.Context, poolID string, target loadbalancer.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return loadbalancer.ErrTargetPoolNotFound
	}

	// Check for duplicate
	for _, t := range pool.Targets {
		if t.Address == target.Address && t.Port == target.Port {
			return loadbalancer.ErrTargetAlreadyRegistered
		}
	}

	newTarget := &loadbalancer.Target{
		ID:           uuid.NewString(),
		Address:      target.Address,
		Port:         target.Port,
		Weight:       target.Weight,
		Status:       loadbalancer.TargetStatusHealthy,
		RegisteredAt: time.Now(),
	}

	if newTarget.Weight <= 0 {
		newTarget.Weight = 1
	}

	pool.Targets = append(pool.Targets, newTarget)
	if bal, ok := m.balancers[poolID]; ok {
		bal.Add(newTarget.ID, newTarget.Weight)
	}
	return nil
}

func (m *Manager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return loadbalancer.ErrTargetPoolNotFound
	}

	for i, t := range pool.Targets {
		if t.ID == targetID {
			pool.Targets = append(pool.Targets[:i], pool.Targets[i+1:]...)
			if bal, ok := m.balancers[poolID]; ok {
				bal.Remove(targetID)
			}
			return nil
		}
	}

	return loadbalancer.ErrTargetNotFound
}

func (m *Manager) GetTargetHealth(ctx context.Context, poolID string) ([]*loadbalancer.Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return nil, loadbalancer.ErrTargetPoolNotFound
	}

	return pool.Targets, nil
}

func (m *Manager) AddRule(ctx context.Context, listenerID string, rule loadbalancer.Rule) (*loadbalancer.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	listener, ok := m.listeners[listenerID]
	if !ok {
		return nil, loadbalancer.ErrListenerNotFound
	}

	newRule := &loadbalancer.Rule{
		ID:           uuid.NewString(),
		Priority:     rule.Priority,
		Conditions:   rule.Conditions,
		TargetPoolID: rule.TargetPoolID,
	}

	listener.Rules = append(listener.Rules, newRule)
	m.rules[listenerID] = listener.Rules

	return newRule, nil
}

func (m *Manager) RemoveRule(ctx context.Context, listenerID, ruleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	listener, ok := m.listeners[listenerID]
	if !ok {
		return loadbalancer.ErrListenerNotFound
	}

	for i, r := range listener.Rules {
		if r.ID == ruleID {
			listener.Rules = append(listener.Rules[:i], listener.Rules[i+1:]...)
			m.rules[listenerID] = listener.Rules
			return nil
		}
	}

	return loadbalancer.ErrRuleNotFound
}

// SelectTarget picks the next target from a pool using pkg/algorithms/loadbalancing
// according to the pool's Algorithm. For least-connections, the selected target's
// connection count is incremented; call ReleaseTarget when the request completes.
func (m *Manager) SelectTarget(ctx context.Context, poolID string) (*loadbalancer.Target, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return nil, loadbalancer.ErrTargetPoolNotFound
	}

	bal, ok := m.balancers[poolID]
	if !ok {
		return nil, loadbalancer.ErrUnsupportedAlgorithm
	}

	nodeID, err := bal.Next(ctx)
	if err != nil {
		if errors.Is(err, loadbalancing.ErrNoNodes) {
			return nil, loadbalancer.ErrNoTargetsAvailable
		}
		return nil, err
	}

	var selected *loadbalancer.Target
	for _, t := range pool.Targets {
		if t.ID == nodeID {
			selected = t
			break
		}
	}
	if selected == nil {
		return nil, loadbalancer.ErrTargetNotFound
	}

	if lc, ok := bal.(*leastconnections.Balancer); ok {
		lc.Inc(nodeID)
	}

	return selected, nil
}

// ReleaseTarget decrements the least-connections count for a previously selected target.
// It is a no-op for other algorithms.
func (m *Manager) ReleaseTarget(ctx context.Context, poolID, targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.targetPools[poolID]; !ok {
		return loadbalancer.ErrTargetPoolNotFound
	}

	bal, ok := m.balancers[poolID]
	if !ok {
		return nil
	}

	if lc, ok := bal.(*leastconnections.Balancer); ok {
		lc.Dec(targetID)
	}
	return nil
}
