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

// Thread-safe public methods

func (h *MinHeap[T]) PushItem(value T, score float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	item := Item[T]{Value: value, Score: score, Index: len(h.items)}
	h.items = append(h.items, item)
	h.up(len(h.items) - 1)
}

func (h *MinHeap[T]) PopItem() (T, float64, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := len(h.items)
	if n == 0 {
		var zero T
		return zero, 0, false
	}

	// Save the root item to return later
	root := h.items[0]

	// Move last item to root
	h.swap(0, n-1)

	// Zero out the last element (which is now the one we want to return)
	// to avoid memory leak if T contains pointers.
	var zero T
	h.items[n-1].Value = zero
	h.items[n-1].Index = -1

	// Shrink the slice
	h.items = h.items[0 : n-1]

	// Restore heap property
	h.down(0, len(h.items))

	return root.Value, root.Score, true
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

// Internal heap operations

func (h *MinHeap[T]) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !h.less(j, i) {
			break
		}
		h.swap(i, j)
		j = i
	}
}

func (h *MinHeap[T]) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !h.less(j, i) {
			break
		}
		h.swap(i, j)
		i = j
	}
	return i > i0
}

func (h *MinHeap[T]) less(i, j int) bool {
	return h.items[i].Score < h.items[j].Score
}

func (h *MinHeap[T]) swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	h.items[i].Index = i
	h.items[j].Index = j
}
