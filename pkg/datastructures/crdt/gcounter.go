package crdt

import (
	"sync"
)

// GCounter is a Grow-only Counter CRDT.
// It supports increments and merges from other replicas.
type GCounter struct {
	id     string
	counts map[string]uint64
	mu     sync.RWMutex
}

func NewGCounter(id string) *GCounter {
	return &GCounter{
		id:     id,
		counts: make(map[string]uint64),
	}
}

// Inc increments the counter for the local node.
func (c *GCounter) Inc(delta uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counts[c.id] += delta
}

// Count returns the total count.
func (c *GCounter) Count() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var total uint64
	for _, v := range c.counts {
		total += v
	}
	return total
}

// Merge merges another GCounter into this one.
// Rule: max(local, remote) for each node ID.
func (c *GCounter) Merge(other *GCounter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for id, val := range other.counts {
		if val > c.counts[id] {
			c.counts[id] = val
		}
	}
}
