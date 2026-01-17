package lru

import (
	"container/list"
	"sync"
)

// Cache is a thread-safe LRU cache.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element
	list     *list.List
	mu       sync.RWMutex
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

// New creates a new LRU cache with the given capacity.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		list:     list.New(),
	}
}

// Get retrieves a value from the cache.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.list.MoveToFront(ent)
		return ent.Value.(*entry[K, V]).value, true
	}

	var zero V
	return zero, false
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.list.MoveToFront(ent)
		ent.Value.(*entry[K, V]).value = value
		return
	}

	ent := c.list.PushFront(&entry[K, V]{key, value})
	c.items[key] = ent

	if c.list.Len() > c.capacity {
		c.removeOldest()
	}
}

// removeOldest removes the oldest item from the cache.
func (c *Cache[K, V]) removeOldest() {
	ent := c.list.Back()
	if ent != nil {
		c.list.Remove(ent)
		kv := ent.Value.(*entry[K, V])
		delete(c.items, kv.key)
	}
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.list.Len()
}

// Clear clears the cache.
func (c *Cache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.list.Init()
	c.items = make(map[K]*list.Element)
}
