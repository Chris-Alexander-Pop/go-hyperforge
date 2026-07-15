// Package maglev implements Google Maglev consistent hashing for load balancing.
//
// Maglev builds a lookup table so each key maps to a backend with minimal disruption
// when backends are added or removed. See: "Maglev: A Fast and Reliable Software
// Network Load Balancer" (Google, NSDI 2016).
package maglev

import (
	"context"
	"hash/fnv"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
)

// DefaultTableSize is a prime table size commonly used for Maglev (65537).
const DefaultTableSize = 65537

// Balancer implements Maglev consistent hashing.
type Balancer struct {
	mu        sync.RWMutex
	nodes     []string
	table     []int // index into nodes; -1 empty during build
	tableSize int
}

// New creates a Maglev balancer. tableSize should be prime; 0 uses DefaultTableSize.
func New(tableSize int, nodes ...string) *Balancer {
	if tableSize <= 0 {
		tableSize = DefaultTableSize
	}
	b := &Balancer{
		nodes:     append([]string(nil), nodes...),
		tableSize: tableSize,
	}
	b.rebuild()
	return b
}

var _ loadbalancing.Balancer = (*Balancer)(nil)

// Next returns the Maglev backend for a constant key of "0" (round-robin-like via
// sequential callers should prefer NextKey). Prefer NextKey for sticky hashing.
func (b *Balancer) Next(ctx context.Context) (string, error) {
	return b.NextKey(ctx, "0")
}

// NextKey selects the backend for key using the Maglev lookup table.
func (b *Balancer) NextKey(ctx context.Context, key string) (string, error) {
	_ = ctx
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.nodes) == 0 || len(b.table) == 0 {
		return "", loadbalancing.ErrNoNodes
	}
	idx := int(hash64(key) % uint64(b.tableSize))
	nodeIdx := b.table[idx]
	if nodeIdx < 0 || nodeIdx >= len(b.nodes) {
		return "", loadbalancing.ErrNoNodes
	}
	return b.nodes[nodeIdx], nil
}

// Add adds a backend and rebuilds the lookup table.
func (b *Balancer) Add(node string, weight int) {
	_ = weight // Maglev treats backends equally; weight reserved for future
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, n := range b.nodes {
		if n == node {
			return
		}
	}
	b.nodes = append(b.nodes, node)
	b.rebuild()
}

// Remove removes a backend and rebuilds the lookup table.
func (b *Balancer) Remove(node string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, n := range b.nodes {
		if n == node {
			b.nodes = append(b.nodes[:i], b.nodes[i+1:]...)
			b.rebuild()
			return
		}
	}
}

func (b *Balancer) rebuild() {
	m := b.tableSize
	n := len(b.nodes)
	table := make([]int, m)
	for i := range table {
		table[i] = -1
	}
	if n == 0 {
		b.table = table
		return
	}

	// permutation[i][j] = (offset_i + j * skip_i) mod M
	offsets := make([]uint64, n)
	skips := make([]uint64, n)
	for i, node := range b.nodes {
		h1, h2 := hash2(node)
		offsets[i] = h1 % uint64(m)
		skips[i] = h2%(uint64(m)-1) + 1
	}

	next := make([]int, n) // next preference index per backend
	filled := 0
	for filled < m {
		for i := 0; i < n && filled < m; i++ {
			c := (offsets[i] + uint64(next[i])*skips[i]) % uint64(m)
			next[i]++
			for table[c] >= 0 {
				c = (offsets[i] + uint64(next[i])*skips[i]) % uint64(m)
				next[i]++
			}
			table[c] = i
			filled++
		}
	}
	b.table = table
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

func hash2(s string) (uint64, uint64) {
	h1 := fnv.New64a()
	_, _ = h1.Write([]byte(s))
	a := h1.Sum64()
	h2 := fnv.New64a()
	_, _ = h2.Write([]byte(s + "#maglev"))
	b := h2.Sum64()
	if b == 0 {
		b = 1
	}
	return a, b
}
