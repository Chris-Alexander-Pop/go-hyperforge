package concurrentmap

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// ShardedMap provides a thread-safe map with reduced lock contention.
// It splits the map into N shards, each with its own RWMutex.
type ShardedMap[K comparable, V any] struct {
	shards     []*shard[K, V]
	shardCount uint32
	shardMask  uint32
}

type shard[K comparable, V any] struct {
	data map[K]V
	mu   *concurrency.SmartRWMutex
}

// New creates a new ShardedMap.
// shardCount is rounded up to the nearest power of 2 for bitwise masking.
func New[K comparable, V any](shardCount int) *ShardedMap[K, V] {
	if shardCount <= 0 {
		shardCount = 32
	}

	// Ensure shardCount is a power of 2
	n := uint32(shardCount)
	// Round up to next power of 2 if not already
	if n&(n-1) != 0 {
		n = 1
		for n < uint32(shardCount) {
			n <<= 1
		}
	}
	shardCount = int(n)

	m := &ShardedMap[K, V]{
		shards:     make([]*shard[K, V], shardCount),
		shardCount: uint32(shardCount),
		shardMask:  uint32(shardCount) - 1,
	}

	for i := 0; i < shardCount; i++ {
		m.shards[i] = &shard[K, V]{
			data: make(map[K]V),
			mu:   concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "ShardedMap"}),
		}
	}

	return m
}

const (
	offset32 = 2166136261
	prime32  = 16777619
)

func (m *ShardedMap[K, V]) getShard(key string) *shard[K, V] {
	// Inline FNV-1a hash implementation
	var hash uint32 = offset32
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= prime32
	}
	// Use bitwise AND for modulo power of 2
	return m.shards[hash&m.shardMask]
}

// Get retrieves a value.
// Note: Key must be string for hashing in this simple implementation.
// For true generic K, we'd need a hash func passed in.
// We'll enforce string keys for now or cast.
func (m *ShardedMap[K, V]) Get(key string) (V, bool) {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	// We need to cast key to K, which works if K is string.
	// This is a limitation of not having a generic Hash(K) function readily available without reflection.
	// For this system design lib, assuming K=string is widespread.
	var k interface{} = key
	val, ok := shard.data[k.(K)]
	return val, ok
}

// Set sets a value.
func (m *ShardedMap[K, V]) Set(key string, value V) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	var k interface{} = key
	shard.data[k.(K)] = value
}

// Delete removes a value.
func (m *ShardedMap[K, V]) Delete(key string) {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	var k interface{} = key
	delete(shard.data, k.(K))
}

// Len returns the total number of items.
func (m *ShardedMap[K, V]) Len() int {
	count := 0
	for _, s := range m.shards {
		s.mu.RLock()
		count += len(s.data)
		s.mu.RUnlock()
	}
	return count
}
