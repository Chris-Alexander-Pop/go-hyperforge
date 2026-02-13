package heap

import (
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
	items []Item[T]
	mu    sync.RWMutex
}

func NewMinHeap[T any]() *MinHeap[T] {
	return &MinHeap[T]{
		items: make([]Item[T], 0),
	}
}

// Internal heap.Interface implementation for compatibility

func (h *MinHeap[T]) Len() int { return len(h.items) }

func (h *MinHeap[T]) Less(i, j int) bool {
	return h.items[i].Score < h.items[j].Score
}

func (h *MinHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].Index = i
	h.items[j].Index = j
}

// Push implements heap.Interface.Push.
// It expects x to be *Item[T] for backward compatibility.
func (h *MinHeap[T]) Push(x any) {
	n := len(h.items)
	item := x.(*Item[T])
	item.Index = n
	h.items = append(h.items, *item)
}

// Pop implements heap.Interface.Pop.
// It returns *Item[T] for backward compatibility.
func (h *MinHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]

	// Zero out the element in the underlying array to prevent memory leak
	// if T contains pointers.
	var zero Item[T]
	old[n-1] = zero

	h.items = old[0 : n-1]
	return &item
}

// Internal optimization: sift up without interface overhead
func (h *MinHeap[T]) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !(h.items[j].Score < h.items[i].Score) {
			break
		}
		h.Swap(j, i)
		j = i
	}
}

// Internal optimization: sift down without interface overhead
func (h *MinHeap[T]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.items[j2].Score < h.items[j1].Score {
			j = j2 // = 2*i + 2  // right child
		}
		if !(h.items[j].Score < h.items[i].Score) {
			break
		}
		h.Swap(i, j)
		i = j
	}
	return i > i0
}

// Thread-safe public methods

func (h *MinHeap[T]) PushItem(value T, score float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Append directly
	n := len(h.items)
	h.items = append(h.items, Item[T]{
		Value: value,
		Score: score,
		Index: n,
	})
	h.up(n)
}

func (h *MinHeap[T]) PopItem() (T, float64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	n := len(h.items)
	if n == 0 {
		var zero T
		return zero, 0, false
	}

	// Swap root with last
	h.Swap(0, n-1)
	// Sift down new root (heap size n-1)
	h.down(0, n-1)

	// Remove last
	item := h.items[n-1]

	// Zero out to prevent memory leak
	var zero Item[T]
	h.items[n-1] = zero

	h.items = h.items[0 : n-1]

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
