package crdt

import (
	"sync"
	"time"
)

// LWWRegister is a Last-Writer-Wins Register CRDT.
// It stores a value and a timestamp.
type LWWRegister[T any] struct {
	value     T
	timestamp int64 // unix nanos
	id        string
	mu        sync.RWMutex
}

func NewLWWRegister[T any](id string, initial T) *LWWRegister[T] {
	return &LWWRegister[T]{
		value:     initial,
		timestamp: time.Now().UnixNano(),
		id:        id,
	}
}

// Set updates the value if the timestamp is newer.
func (r *LWWRegister[T]) Set(value T, ts int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If timestamp is strictly greater, overwrite.
	// Tie-breaking: usually prefer higher ID or arbitrary deterministic rule.
	// Here we just ignore if equal for simplicity, or overwrite (Last Writer).
	if ts > r.timestamp {
		r.value = value
		r.timestamp = ts
	}
}

// Get returns the current value.
func (r *LWWRegister[T]) Get() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

// Merge merges another register.
func (r *LWWRegister[T]) Merge(other *LWWRegister[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	if other.timestamp > r.timestamp {
		r.value = other.value
		r.timestamp = other.timestamp
	}
}
