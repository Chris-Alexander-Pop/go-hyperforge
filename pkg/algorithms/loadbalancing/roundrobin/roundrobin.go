package roundrobin

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
)

// Balancer implementation for Round Robin.
// Cycles through nodes sequentially.
type Balancer struct {
	nodes []string
	count uint64
	mu    sync.RWMutex
}

// New creates a new RoundRobin balancer.
func New(nodes ...string) *Balancer {
	return &Balancer{
		nodes: nodes,
	}
}

// Next returns the next node.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.nodes)
	if n == 0 {
		return "", loadbalancing.ErrNoNodes
	}

	// Atomic increment for thread safety without full lock
	count := atomic.AddUint64(&b.count, 1)
	return b.nodes[(count-1)%uint64(n)], nil
}

// Add adds a new node to the pool.
func (b *Balancer) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes = append(b.nodes, node)
}

// Remove removes a node from the pool.
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
