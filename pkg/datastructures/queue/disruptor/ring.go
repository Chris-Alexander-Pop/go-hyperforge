package disruptor

import (
	"runtime"
	"sync/atomic"
)

const (
	padding = 64 // Cache line padding to prevent false sharing
)

// RingBuffer is a high-performance lock-free ring buffer.
type RingBuffer[T any] struct {
	buffer []T
	mask   uint64
	cursor uint64 // Writer cursor
	_      [padding]byte
	gate   uint64 // Reader gating cursor
}

func New[T any](size uint64) *RingBuffer[T] {
	// Size must be power of 2
	if size == 0 || (size&(size-1)) != 0 {
		size = 1024 // Default
	}

	return &RingBuffer[T]{
		buffer: make([]T, size),
		mask:   size - 1,
	}
}

// Publish reserves a slot and runs the callback to fill it.
func (rb *RingBuffer[T]) Publish(producer func(*T)) {
	// Simple Single-Producer logic.
	// For Multi-Producer, need CAS on cursor.

	current := atomic.LoadUint64(&rb.cursor)
	next := current + 1

	// Check wrap around against slow readers (gate)
	// wrapPoint := next - size.
	// if wrapPoint > gate, spin.
	// Only necessary if preventing overwrite.
	// Disruptor assumes gating.

	idx := next & rb.mask
	producer(&rb.buffer[idx])

	atomic.StoreUint64(&rb.cursor, next)
}

// Consume waits for availability and consumes.
func (rb *RingBuffer[T]) Consume(consumer func(T)) {
	// Simple Single-Consumer logic.

	target := atomic.LoadUint64(&rb.gate) + 1

	// Spin wait for cursor to reach target
	for target > atomic.LoadUint64(&rb.cursor) {
		runtime.Gosched()
	}

	// Read
	idx := target & rb.mask
	consumer(rb.buffer[idx])

	atomic.StoreUint64(&rb.gate, target)
}

// Note: This is an extremely simplified schematic of the LMAX Disruptor.
// Real disruptor deals with Barriers, multiple dependencies, and memory barriers extensively.
