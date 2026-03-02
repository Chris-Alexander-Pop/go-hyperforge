package concurrentmap

import (
	"strconv"
	"testing"
)

func BenchmarkGetShard(b *testing.B) {
	m := New[string, int](32)
	key := "test_key_benchmark"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.getShard(key)
	}
}

func BenchmarkSet(b *testing.B) {
	m := New[string, int](32)
	key := "test_key"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(key, i)
	}
}

func BenchmarkGet(b *testing.B) {
	m := New[string, int](32)
	key := "test_key"
	m.Set(key, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(key)
	}
}

func BenchmarkSetMany(b *testing.B) {
	m := New[string, int](32)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(keys[i%1000], i)
	}
}
