package concurrency

import (
	"hash/fnv"
	"sync"
)

const shardCount = 64

type ShardedMap[K comparable, V any] struct {
	shards []*shard[K, V]
}

type shard[K comparable, V any] struct {
	sync.RWMutex
	data map[K]V
}

func NewShardedMap[K comparable, V any]() *ShardedMap[K, V] {
	m := &ShardedMap[K, V]{
		shards: make([]*shard[K, V], shardCount),
	}
	for i := 0; i < shardCount; i++ {
		m.shards[i] = &shard[K, V]{
			data: make(map[K]V),
		}
	}
	return m
}

func (m *ShardedMap[K, V]) getShard(key string) *shard[K, V] {
	h := fnv.New32a()
	h.Write([]byte(key))
	idx := h.Sum32() % shardCount
	return m.shards[idx]
}

// Get helper assumes key is string for hashing.
// Limitation of generics in Go: cannot hash generic 'comparable' easily without reflection.
// I will specialize this map to `string` keys for maximum performance to avoid reflection.
// Returning to: type ShardedMap struct... keys are strings.
// Or forcing K to be Stringer?
// simpler: ShardedMap is map[string]interface{} or map[string]V.
// I'll make it ShardedMap[V any] and keys are strings.

type ShardedMapString[V any] struct {
	shards []*shardString[V]
}
type shardString[V any] struct {
	sync.RWMutex
	data map[string]V
}

func NewShardedMapString[V any]() *ShardedMapString[V] {
	m := &ShardedMapString[V]{
		shards: make([]*shardString[V], shardCount),
	}
	for i := 0; i < shardCount; i++ {
		m.shards[i] = &shardString[V]{
			data: make(map[string]V),
		}
	}
	return m
}

func (m *ShardedMapString[V]) getShard(key string) *shardString[V] {
	h := fnv.New32a()
	h.Write([]byte(key))
	return m.shards[uint(h.Sum32())%shardCount]
}

func (m *ShardedMapString[V]) Set(key string, value V) {
	shard := m.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	shard.data[key] = value
}

func (m *ShardedMapString[V]) Get(key string) (V, bool) {
	shard := m.getShard(key)
	shard.RLock()
	defer shard.RUnlock()
	val, ok := shard.data[key]
	return val, ok
}

func (m *ShardedMapString[V]) Delete(key string) {
	shard := m.getShard(key)
	shard.Lock()
	defer shard.Unlock()
	delete(shard.data, key)
}
