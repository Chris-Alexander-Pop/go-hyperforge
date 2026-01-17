package set

import "sync"

// Set is a generic thread-safe set.
type Set[T comparable] struct {
	m  map[T]struct{}
	mu sync.RWMutex
}

// New creates a new Set.
func New[T comparable](items ...T) *Set[T] {
	s := &Set[T]{
		m: make(map[T]struct{}),
	}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add adds an item.
func (s *Set[T]) Add(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[item] = struct{}{}
}

// Remove removes an item.
func (s *Set[T]) Remove(item T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, item)
}

// Contains checks if item exists.
func (s *Set[T]) Contains(item T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.m[item]
	return exists
}

// Len returns the number of items.
func (s *Set[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.m)
}

// List returns a slice of all items.
func (s *Set[T]) List() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]T, 0, len(s.m))
	for k := range s.m {
		list = append(list, k)
	}
	return list
}

// Union returns a new set with elements from both sets.
func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := New[T]()

	s.mu.RLock()
	for k := range s.m {
		result.Add(k)
	}
	s.mu.RUnlock()

	other.mu.RLock()
	for k := range other.m {
		result.Add(k)
	}
	other.mu.RUnlock()

	return result
}

// Intersection returns a new set with elements common to both.
func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := New[T]()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for k := range s.m {
		if other.Contains(k) {
			result.Add(k)
		}
	}
	return result
}
