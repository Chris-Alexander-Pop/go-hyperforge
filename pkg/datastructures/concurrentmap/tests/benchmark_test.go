package concurrentmap_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/concurrentmap"
)

func BenchmarkConcurrentMap_Set(b *testing.B) {
	m := concurrentmap.New[string, int](32)
	keys := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(keys[i], i)
	}
}

func BenchmarkConcurrentMap_Get(b *testing.B) {
	m := concurrentmap.New[string, int](32)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
		m.Set(keys[i], i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(keys[i%1000])
	}
}
