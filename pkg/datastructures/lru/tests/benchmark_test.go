package lru_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/lru"
)

var keys []string

func init() {
	keys = make([]string, 10000)
	for i := 0; i < 10000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
}

func BenchmarkLRU_Set(b *testing.B) {
	cache := lru.New[string, int](1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(keys[i%10000], i)
	}
}

func BenchmarkLRU_Get(b *testing.B) {
	cache := lru.New[string, int](1000)
	for i := 0; i < 1000; i++ {
		cache.Set(keys[i], i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(keys[i%1000])
	}
}
