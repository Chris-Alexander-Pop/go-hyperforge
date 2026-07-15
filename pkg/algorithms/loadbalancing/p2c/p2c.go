// Package p2c implements Power-of-Two-Choices load balancing.
//
// Each Next() picks two random backends and returns the one with fewer active
// connections (Inc/Dec instrument like leastconnections).
package p2c

import (
	"context"
	"math/rand"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Balancer implements power-of-two-choices selection.
type Balancer struct {
	mu    *concurrency.SmartRWMutex
	nodes map[string]int64
	list  []string
	rng   *rand.Rand
}

// New creates a P2C balancer.
func New(nodes ...string) *Balancer {
	m := make(map[string]int64, len(nodes))
	list := make([]string, 0, len(nodes))
	for _, n := range nodes {
		if _, ok := m[n]; ok {
			continue
		}
		m[n] = 0
		list = append(list, n)
	}
	return &Balancer{
		nodes: m,
		list:  list,
		rng:   rand.New(rand.NewSource(1)), // deterministic default; tests may replace via seed
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "lb-p2c"}),
	}
}

// NewWithSeed creates a P2C balancer with an explicit RNG seed.
func NewWithSeed(seed int64, nodes ...string) *Balancer {
	b := New(nodes...)
	b.rng = rand.New(rand.NewSource(seed))
	return b
}

var _ loadbalancing.Balancer = (*Balancer)(nil)

// Next picks two candidates and returns the less-loaded one.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	_ = ctx
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(b.list)
	if n == 0 {
		return "", loadbalancing.ErrNoNodes
	}
	if n == 1 {
		return b.list[0], nil
	}
	i := b.rng.Intn(n)
	j := b.rng.Intn(n - 1)
	if j >= i {
		j++
	}
	a, c := b.list[i], b.list[j]
	if b.nodes[a] <= b.nodes[c] {
		return a, nil
	}
	return c, nil
}

// Inc increments active connections for a node.
func (b *Balancer) Inc(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; ok {
		b.nodes[node]++
	}
}

// Dec decrements active connections for a node.
func (b *Balancer) Dec(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if c, ok := b.nodes[node]; ok && c > 0 {
		b.nodes[node]--
	}
}

// Load returns the active connection count for a node (0 if unknown).
func (b *Balancer) Load(node string) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nodes[node]
}

// Add adds a node.
func (b *Balancer) Add(node string, weight int) {
	_ = weight
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; ok {
		return
	}
	b.nodes[node] = 0
	b.list = append(b.list, node)
}

// Remove removes a node.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.nodes[node]; !ok {
		return
	}
	delete(b.nodes, node)
	for i, n := range b.list {
		if n == node {
			b.list = append(b.list[:i], b.list[i+1:]...)
			return
		}
	}
}
