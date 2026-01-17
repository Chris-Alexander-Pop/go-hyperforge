package ring

import (
	"errors"
	"sync"
)

var ErrBufferFull = errors.New("ring buffer is full")
var ErrBufferEmpty = errors.New("ring buffer is empty")

// Buffer is a fixed-size circular buffer.
type Buffer[T any] struct {
	buf      []T
	head     int
	tail     int
	count    int
	size     int
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond
}

// New creates a new Ring Buffer.
func New[T any](size int) *Buffer[T] {
	if size <= 0 {
		size = 1
	}
	b := &Buffer[T]{
		buf:  make([]T, size),
		size: size,
	}
	b.notEmpty = sync.NewCond(&b.mu)
	b.notFull = sync.NewCond(&b.mu)
	return b
}

// Enqueue adds an item. Blocks if full (use TryEnqueue for non-blocking).
func (b *Buffer[T]) Enqueue(item T) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for b.count == b.size {
		b.notFull.Wait()
	}

	b.buf[b.tail] = item
	b.tail = (b.tail + 1) % b.size
	b.count++
	b.notEmpty.Signal()
}

// TryEnqueue adds an item if space is available.
func (b *Buffer[T]) TryEnqueue(item T) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == b.size {
		return ErrBufferFull
	}

	b.buf[b.tail] = item
	b.tail = (b.tail + 1) % b.size
	b.count++
	b.notEmpty.Signal()
	return nil
}

// Dequeue removes an item. Blocks if empty.
func (b *Buffer[T]) Dequeue() T {
	b.mu.Lock()
	defer b.mu.Unlock()

	for b.count == 0 {
		b.notEmpty.Wait()
	}

	item := b.buf[b.head]
	// zero out for GC if T is a pointer type, though generics makes this tricky.
	// We'll leave it for now.
	b.head = (b.head + 1) % b.size
	b.count--
	b.notFull.Signal()
	return item
}

// TryDequeue removes an item if available.
func (b *Buffer[T]) TryDequeue() (T, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.count == 0 {
		var zero T
		return zero, ErrBufferEmpty
	}

	item := b.buf[b.head]
	b.head = (b.head + 1) % b.size
	b.count--
	b.notFull.Signal()
	return item, nil
}

// Len returns the number of items.
func (b *Buffer[T]) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// Cap returns the capacity.
func (b *Buffer[T]) Cap() int {
	return b.size
}
