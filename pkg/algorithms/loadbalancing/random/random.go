package random

import (
	"context"
	"math/rand"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Balancer implementation for Random selection.
type Balancer struct {
	nodes []string
	mu    *concurrency.SmartRWMutex
}

// New creates a new Random balancer.
func New(nodes ...string) *Balancer {
	return &Balancer{
		nodes: nodes,
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "lb-random"}),
	}
}

// Next returns a random node.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.nodes)
	if n == 0 {
		return "", loadbalancing.ErrNoNodes
	}

	return b.nodes[rand.Intn(n)], nil
}

// Add adds a node.
func (b *Balancer) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes = append(b.nodes, node)
}

// Remove removes a node.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, n := range b.nodes {
		if n == node {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			return
		}
	}
}
