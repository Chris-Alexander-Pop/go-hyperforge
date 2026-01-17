package heap

import (
	"container/heap"
	"sync"
)

// Item represents an item in the heap with a score/priority.
type Item[T any] struct {
	Value T
	Score float64
	Index int // Internal index for heap
}

// MinHeap implements a min-heap (lowest score at root).
type MinHeap[T any] struct {
	items []*Item[T]
	mu    sync.RWMutex
}

func NewMinHeap[T any]() *MinHeap[T] {
	h := &MinHeap[T]{
		items: make([]*Item[T], 0),
	}
	heap.Init(h)
	return h
}

// Internal heap.Interface implementation
func (h *MinHeap[T]) Len() int           { return len(h.items) }
func (h *MinHeap[T]) Less(i, j int) bool { return h.items[i].Score < h.items[j].Score }
func (h *MinHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].Index = i
	h.items[j].Index = j
}
func (h *MinHeap[T]) Push(x interface{}) {
	n := len(h.items)
	item := x.(*Item[T])
	item.Index = n
	h.items = append(h.items, item)
}
func (h *MinHeap[T]) Pop() interface{} {
	old := h.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // Avoid memory leak
	item.Index = -1
	h.items = old[0 : n-1]
	return item
}

// Thread-safe public methods

func (h *MinHeap[T]) PushItem(value T, score float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	heap.Push(h, &Item[T]{Value: value, Score: score})
}

func (h *MinHeap[T]) PopItem() (T, float64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.items) == 0 {
		var zero T
		return zero, 0, false
	}
	item := heap.Pop(h).(*Item[T])
	return item.Value, item.Score, true
}

func (h *MinHeap[T]) Peek() (T, float64, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.items) == 0 {
		var zero T
		return zero, 0, false
	}
	item := h.items[0]
	return item.Value, item.Score, true
}

func (h *MinHeap[T]) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.items)
}
