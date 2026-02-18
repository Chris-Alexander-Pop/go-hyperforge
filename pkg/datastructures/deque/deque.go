package deque

import (
	"math/bits"
	"sync"
)

// Deque is a generic double-ended queue.
// Implemented using a slice with head/tail tracking or just simple slice manip (less efficient).
// For efficiency, standard Deque uses linked list or ring buffer.
// Let's use a slice-based approach with resizing/re-centering if needed, or just container/list wrapper?
// The user prompt asked for "LeetCode" style, usually meaning O(1) ops.
// A simpler but efficient implementation is a doubly-linked list (which we have in standard lib but not generic).
// We'll implement a Ring Buffer based Deque for performance.

type Deque[T any] struct {
	buf    []T
	head   int
	tail   int
	count  int
	minCap int
	mu     sync.RWMutex
}

func New[T any](initialCap int) *Deque[T] {
	if initialCap < 1 {
		initialCap = 16
	} else {
		// Round up to the next power of 2 to ensure efficient bitwise AND operations
		// for circular buffer indexing.
		initialCap = 1 << bits.Len(uint(initialCap-1))
	}
	return &Deque[T]{
		buf:    make([]T, initialCap),
		minCap: initialCap,
	}
}

// PushBack adds to the back.
func (d *Deque[T]) PushBack(val T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.growIfFull()
	d.buf[d.tail] = val
	d.tail = (d.tail + 1) & (len(d.buf) - 1)
	d.count++
}

// PushFront adds to the front.
func (d *Deque[T]) PushFront(val T) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.growIfFull()
	d.head = (d.head - 1) & (len(d.buf) - 1)
	d.buf[d.head] = val
	d.count++
}

// PopBack removes from back.
func (d *Deque[T]) PopBack() (T, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.count == 0 {
		var zero T
		return zero, false
	}
	d.tail = (d.tail - 1) & (len(d.buf) - 1)
	val := d.buf[d.tail]

	// Avoid memory leaks
	var zero T
	d.buf[d.tail] = zero

	d.count--
	return val, true
}

// PopFront removes from front.
func (d *Deque[T]) PopFront() (T, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.count == 0 {
		var zero T
		return zero, false
	}
	val := d.buf[d.head]

	// Avoid memory leaks
	var zero T
	d.buf[d.head] = zero

	d.head = (d.head + 1) & (len(d.buf) - 1)
	d.count--
	return val, true
}

func (d *Deque[T]) growIfFull() {
	if d.count == len(d.buf) {
		d.resize(len(d.buf) * 2)
	}
}

func (d *Deque[T]) resize(newSize int) {
	newBuf := make([]T, newSize)
	if d.tail > d.head {
		copy(newBuf, d.buf[d.head:d.tail])
	} else {
		n := copy(newBuf, d.buf[d.head:])
		copy(newBuf[n:], d.buf[:d.tail])
	}
	d.head = 0
	d.tail = d.count
	d.buf = newBuf
}

func (d *Deque[T]) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.count
}
