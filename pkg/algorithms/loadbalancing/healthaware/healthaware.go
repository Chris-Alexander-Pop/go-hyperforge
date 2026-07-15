// Package healthaware wraps a Balancer and skips nodes reported unhealthy.
package healthaware

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Checker reports whether a node should receive traffic.
type Checker interface {
	Healthy(ctx context.Context, node string) bool
}

// CheckerFunc adapts a function to Checker.
type CheckerFunc func(ctx context.Context, node string) bool

// Healthy implements Checker.
func (f CheckerFunc) Healthy(ctx context.Context, node string) bool {
	return f(ctx, node)
}

// Balancer wraps an inner loadbalancing.Balancer and skips unhealthy nodes.
type Balancer struct {
	inner   loadbalancing.Balancer
	checker Checker
	mu      *concurrency.SmartRWMutex
}

// New wraps inner with health checks. checker may be nil (all nodes healthy).
func New(inner loadbalancing.Balancer, checker Checker) *Balancer {
	return &Balancer{
		inner:   inner,
		checker: checker,
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "lb-healthaware"}),
	}
}

var _ loadbalancing.Balancer = (*Balancer)(nil)

// Next returns the next healthy node. If a full selection cycle yields only
// unhealthy nodes, ErrNoNodes is returned.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	if b.inner == nil {
		return "", loadbalancing.ErrNoNodes
	}
	seen := make(map[string]struct{})
	for {
		node, err := b.inner.Next(ctx)
		if err != nil {
			return "", err
		}
		if b.isHealthy(ctx, node) {
			return node, nil
		}
		if _, ok := seen[node]; ok {
			return "", loadbalancing.ErrNoNodes
		}
		seen[node] = struct{}{}
		if len(seen) > 4096 {
			return "", loadbalancing.ErrNoNodes
		}
	}
}

func (b *Balancer) isHealthy(ctx context.Context, node string) bool {
	b.mu.RLock()
	c := b.checker
	b.mu.RUnlock()
	if c == nil {
		return true
	}
	return c.Healthy(ctx, node)
}

// Add delegates to the inner balancer.
func (b *Balancer) Add(node string, weight int) {
	if b.inner != nil {
		b.inner.Add(node, weight)
	}
}

// Remove delegates to the inner balancer.
func (b *Balancer) Remove(node string) {
	if b.inner != nil {
		b.inner.Remove(node)
	}
}

// SetChecker replaces the health checker (nil = all healthy).
func (b *Balancer) SetChecker(c Checker) {
	b.mu.Lock()
	b.checker = c
	b.mu.Unlock()
}
