package heap_test

import (
	stdheap "container/heap"
	"testing"
	myheap "github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
)

func TestHeapInterfaceCompatibility(t *testing.T) {
	h := myheap.NewMinHeap[int]()
	// Verify it implements heap.Interface
	var _ stdheap.Interface = h

	// Use heap.Push
	stdheap.Push(h, &myheap.Item[int]{Value: 10, Score: 10.0})
	stdheap.Push(h, &myheap.Item[int]{Value: 5, Score: 5.0})
	stdheap.Push(h, &myheap.Item[int]{Value: 20, Score: 20.0})

	if h.Len() != 3 {
		t.Errorf("Expected length 3, got %d", h.Len())
	}

	// Use heap.Pop
	item := stdheap.Pop(h).(*myheap.Item[int])
	if item.Value != 5 {
		t.Errorf("Expected popped value 5, got %v", item.Value)
	}

	// Verify internal state is consistent
	val, score, ok := h.Peek()
	if !ok || val != 10 || score != 10.0 {
		t.Errorf("Expected peek 10, got %v with score %v", val, score)
	}
}
