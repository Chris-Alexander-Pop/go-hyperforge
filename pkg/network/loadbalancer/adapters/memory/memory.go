package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/network/loadbalancer"
	"github.com/google/uuid"
)

// Manager implements an in-memory load balancer manager for testing.
type Manager struct {
	mu            sync.RWMutex
	loadBalancers map[string]*loadbalancer.LoadBalancer
	listeners     map[string]*loadbalancer.Listener // listenerID -> listener
	targetPools   map[string]*loadbalancer.TargetPool
	rules         map[string][]*loadbalancer.Rule // listenerID -> rules
	config        loadbalancer.Config
}

// New creates a new in-memory load balancer manager.
func New() *Manager {
	return &Manager{
		loadBalancers: make(map[string]*loadbalancer.LoadBalancer),
		listeners:     make(map[string]*loadbalancer.Listener),
		targetPools:   make(map[string]*loadbalancer.TargetPool),
		rules:         make(map[string][]*loadbalancer.Rule),
		config:        loadbalancer.Config{},
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
		return nil, errors.NotFound("load balancer not found", nil)
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
		return errors.NotFound("load balancer not found", nil)
	}

	if len(lb.Listeners) > 0 {
		return errors.Conflict("load balancer has active listeners", nil)
	}

	delete(m.loadBalancers, id)
	return nil
}

func (m *Manager) CreateListener(ctx context.Context, opts loadbalancer.CreateListenerOptions) (*loadbalancer.Listener, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	lb, ok := m.loadBalancers[opts.LoadBalancerID]
	if !ok {
		return nil, errors.NotFound("load balancer not found", nil)
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
		return errors.NotFound("load balancer not found", nil)
	}

	if _, ok := m.listeners[listenerID]; !ok {
		return errors.NotFound("listener not found", nil)
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

	pool := &loadbalancer.TargetPool{
		ID:          uuid.NewString(),
		Name:        opts.Name,
		Protocol:    opts.Protocol,
		Port:        opts.Port,
		Algorithm:   opts.Algorithm,
		HealthCheck: opts.HealthCheck,
		Targets:     []*loadbalancer.Target{},
		Tags:        opts.Tags,
		CreatedAt:   time.Now(),
	}

	if pool.Algorithm == "" {
		pool.Algorithm = loadbalancer.AlgorithmRoundRobin
	}

	m.targetPools[pool.ID] = pool
	return pool, nil
}

func (m *Manager) GetTargetPool(ctx context.Context, id string) (*loadbalancer.TargetPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.targetPools[id]
	if !ok {
		return nil, errors.NotFound("target pool not found", nil)
	}

	return pool, nil
}

func (m *Manager) DeleteTargetPool(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[id]
	if !ok {
		return errors.NotFound("target pool not found", nil)
	}

	if len(pool.Targets) > 0 {
		return errors.Conflict("target pool has registered targets", nil)
	}

	delete(m.targetPools, id)
	return nil
}

func (m *Manager) AddTarget(ctx context.Context, poolID string, target loadbalancer.Target) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return errors.NotFound("target pool not found", nil)
	}

	// Check for duplicate
	for _, t := range pool.Targets {
		if t.Address == target.Address && t.Port == target.Port {
			return errors.Conflict("target already registered", nil)
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
	return nil
}

func (m *Manager) RemoveTarget(ctx context.Context, poolID, targetID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return errors.NotFound("target pool not found", nil)
	}

	for i, t := range pool.Targets {
		if t.ID == targetID {
			pool.Targets = append(pool.Targets[:i], pool.Targets[i+1:]...)
			return nil
		}
	}

	return errors.NotFound("target not found", nil)
}

func (m *Manager) GetTargetHealth(ctx context.Context, poolID string) ([]*loadbalancer.Target, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.targetPools[poolID]
	if !ok {
		return nil, errors.NotFound("target pool not found", nil)
	}

	return pool.Targets, nil
}

func (m *Manager) AddRule(ctx context.Context, listenerID string, rule loadbalancer.Rule) (*loadbalancer.Rule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	listener, ok := m.listeners[listenerID]
	if !ok {
		return nil, errors.NotFound("listener not found", nil)
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
		return errors.NotFound("listener not found", nil)
	}

	for i, r := range listener.Rules {
		if r.ID == ruleID {
			listener.Rules = append(listener.Rules[:i], listener.Rules[i+1:]...)
			m.rules[listenerID] = listener.Rules
			return nil
		}
	}

	return errors.NotFound("rule not found", nil)
}
