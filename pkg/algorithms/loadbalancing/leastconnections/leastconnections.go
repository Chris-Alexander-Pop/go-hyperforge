package leastconnections

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Balancer selects the node with the fewest active connections.
// It requires manual instrumentation (Inc/Dec).
type Balancer struct {
	nodes map[string]int64 // node -> active count
	mu    *concurrency.SmartRWMutex
}

// New creates a new LeastConnections balancer.
func New(nodes ...string) *Balancer {
	m := make(map[string]int64)
	for _, n := range nodes {
		m[n] = 0
	}
	return &Balancer{
		nodes: m,
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "lb-leastconnections"}),
	}
}

// Next returns the node with the least connections.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.nodes) == 0 {
		return "", loadbalancing.ErrNoNodes
	}

	var bestNode string
	var minConns int64 = -1

	for node, conns := range b.nodes {
		if minConns == -1 || conns < minConns {
			minConns = conns
			bestNode = node
		}
	}

	return bestNode, nil
}

// Inc increments the connection count for a node.
// Call this when a request starts.
func (b *Balancer) Inc(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; ok {
		b.nodes[node]++
	}
}

// Dec decrements the connection count for a node.
// Call this when a request ends.
func (b *Balancer) Dec(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if count, ok := b.nodes[node]; ok && count > 0 {
		b.nodes[node]--
	}
}

// Add adds a node.
func (b *Balancer) Add(node string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; !ok {
		b.nodes[node] = 0
	}
}

// Remove removes a node.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.nodes, node)
}
