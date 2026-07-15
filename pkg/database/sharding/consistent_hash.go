package sharding

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/consistenthash/ring"
)

// Strategy defines the interface for sharding strategies
type Strategy interface {
	GetShard(key string) string
	AddShard(shardID string)
	RemoveShard(shardID string)
}

// ConsistentHash implements a consistent hashing ring
// Wrapper around generic algorithm
type ConsistentHash struct {
	ring *ring.Ring
}

// NewConsistentHash creates a new ConsistentHash strategy
func NewConsistentHash(replicas int, shards []string) *ConsistentHash {
	return &ConsistentHash{
		ring: ring.New(replicas, shards),
	}
}

func (m *ConsistentHash) AddShard(shardID string) {
	m.ring.Add(shardID)
}

func (m *ConsistentHash) RemoveShard(shardID string) {
	m.ring.Remove(shardID)
}

func (m *ConsistentHash) GetShard(key string) string {
	return m.ring.Get(key)
}
