package heap_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
)

func BenchmarkHeapPush(b *testing.B) {
	h := heap.NewMinHeap[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PushItem(i, float64(i))
	}
}

func BenchmarkHeapPushPop(b *testing.B) {
	h := heap.NewMinHeap[int]()
	// Pre-fill
	for i := 0; i < 1000; i++ {
		h.PushItem(i, float64(i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PushItem(i, float64(i))
		h.PopItem()
	}
}
