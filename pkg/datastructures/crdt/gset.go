package crdt

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// GSet is a grow-only set CRDT.
// Elements can be added but never removed; Merge is a set-union.
type GSet[T comparable] struct {
	items map[T]struct{}
	mu    *concurrency.SmartRWMutex
}

// NewGSet creates an empty grow-only set.
func NewGSet[T comparable]() *GSet[T] {
	return &GSet[T]{
		items: make(map[T]struct{}),
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "GSet"}),
	}
}

// Add inserts an element into the set.
func (s *GSet[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

// Contains reports whether item is in the set.
func (s *GSet[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.items[item]
	return ok
}

// Size returns the number of elements.
func (s *GSet[T]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Elements returns a snapshot of the set contents (order undefined).
func (s *GSet[T]) Elements() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]T, 0, len(s.items))
	for item := range s.items {
		out = append(out, item)
	}
	return out
}

// Merge unions other into this set (grow-only).
func (s *GSet[T]) Merge(other *GSet[T]) {
	if other == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()
	for item := range other.items {
		s.items[item] = struct{}{}
	}
}
