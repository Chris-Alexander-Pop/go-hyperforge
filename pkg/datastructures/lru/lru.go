package lru

import (
	"sync"
)

// Cache is a thread-safe LRU cache.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*entry[K, V]
	head     *entry[K, V]
	tail     *entry[K, V]
	size     int
	mu       sync.RWMutex
}

type entry[K comparable, V any] struct {
	key   K
	value V
	prev  *entry[K, V]
	next  *entry[K, V]
}

// New creates a new LRU cache with the given capacity.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]*entry[K, V]),
	}
}

// Get retrieves a value from the cache.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.moveToFront(ent)
		return ent.value, true
	}

	var zero V
	return zero, false
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.moveToFront(ent)
		ent.value = value
		return
	}

	ent := &entry[K, V]{key: key, value: value}
	c.pushFront(ent)
	c.items[key] = ent

	if c.size > c.capacity {
		c.removeOldest()
	}
}

// removeOldest removes the oldest item from the cache.
func (c *Cache[K, V]) removeOldest() {
	if c.tail != nil {
		ent := c.tail
		c.remove(ent)
		delete(c.items, ent.key)
	}
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}

// Clear clears the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.head = nil
	c.tail = nil
	c.size = 0
	c.items = make(map[K]*entry[K, V])
}

// pushFront adds a node to the front of the list.
func (c *Cache[K, V]) pushFront(e *entry[K, V]) {
	if c.head == nil {
		c.head = e
		c.tail = e
		e.prev = nil
		e.next = nil
	} else {
		e.next = c.head
		e.prev = nil
		c.head.prev = e
		c.head = e
	}
	c.size++
}

// moveToFront moves a node to the front of the list.
func (c *Cache[K, V]) moveToFront(e *entry[K, V]) {
	if c.head == e {
		return
	}

	// Unlink e
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == c.tail {
		c.tail = e.prev
	}

	// Link e at front
	e.next = c.head
	e.prev = nil
	if c.head != nil {
		c.head.prev = e
	}
	c.head = e
}

// remove removes a node from the list.
func (c *Cache[K, V]) remove(e *entry[K, V]) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		c.head = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	} else {
		c.tail = e.prev
	}
	e.next = nil // avoid memory leaks
	e.prev = nil // avoid memory leaks
	c.size--
}
