package heap_test

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
	"testing"
)

func BenchmarkMinHeap_PushPop(b *testing.B) {
	h := heap.NewMinHeap[int]()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PushItem(i, float64(i))
		if h.Size() >= 1000 {
			for h.Size() > 0 {
				h.PopItem()
			}
		}
	}
}
