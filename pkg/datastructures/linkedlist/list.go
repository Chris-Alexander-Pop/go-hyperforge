package linkedlist

import (
	"container/list"
	"sync"
)

// List wraps container/list with thread safety.
type List[T any] struct {
	l  *list.List
	mu sync.RWMutex
}

func New[T any]() *List[T] {
	return &List[T]{
		l: list.New(),
	}
}

// PushBack adds an element to the back of the list.
func (l *List[T]) PushBack(v T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.PushBack(v)
}

// PushFront adds an element to the front of the list.
func (l *List[T]) PushFront(v T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.l.PushFront(v)
}

// PopFront removes and returns the front element.
func (l *List[T]) PopFront() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.l.Front()
	if elem == nil {
		var zero T
		return zero, false
	}
	l.l.Remove(elem)
	return elem.Value.(T), true
}

// PopBack removes and returns the back element.
func (l *List[T]) PopBack() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	elem := l.l.Back()
	if elem == nil {
		var zero T
		return zero, false
	}
	l.l.Remove(elem)
	return elem.Value.(T), true
}

// Len returns the number of elements.
func (l *List[T]) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.l.Len()
}
