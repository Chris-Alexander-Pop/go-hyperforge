package concurrentmap

import (
	"fmt"
	"testing"
)

// BenchmarkShardedMap_Get measures the performance of retrieving an item.
func BenchmarkShardedMap_Get(b *testing.B) {
	m := New[string, int](32)
	key := "test_key"
	m.Set(key, 100)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(key)
	}
}

// BenchmarkShardedMap_Set measures the performance of setting an item.
func BenchmarkShardedMap_Set(b *testing.B) {
	m := New[string, int](32)
	key := "test_key"

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Set(key, i)
	}
}

// BenchmarkShardedMap_ShardSelection focuses on the overhead of hashing and shard selection.
func BenchmarkShardedMap_ShardSelection(b *testing.B) {
	m := New[string, int](32)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		m.Set(keys[i], i)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := keys[i%1000]
		m.Get(key)
	}
}
