package lfu

import (
	"container/list"
	"sync"
)

// Cache is a thread-safe LFU cache.
// Implementation complexity is O(1) for Get and Set using frequency lists.
type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]*list.Element // key -> element in standard list (node)
	freqs    map[int]*list.List  // frequency -> list of nodes
	minFreq  int
	mu       sync.RWMutex
}

type entry[K comparable, V any] struct {
	key   K
	value V
	freq  int
}

// New creates a new LFU cache with the given capacity.
func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
		freqs:    make(map[int]*list.List),
		minFreq:  0,
	}
}

// Get retrieves a value from the cache.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		val := ent.Value.(*entry[K, V])
		c.incrementFreq(ent, val)
		return val.value, true
	}

	var zero V
	return zero, false
}

// Set adds a value to the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		val := ent.Value.(*entry[K, V])
		val.value = value
		c.incrementFreq(ent, val)
		return
	}

	if len(c.items) >= c.capacity {
		c.evict()
	}

	// New items start with freq 1
	val := &entry[K, V]{key: key, value: value, freq: 1}
	if c.freqs[1] == nil {
		c.freqs[1] = list.New()
	}
	ent := c.freqs[1].PushFront(val)
	c.items[key] = ent
	c.minFreq = 1
}

func (c *Cache[K, V]) incrementFreq(ent *list.Element, val *entry[K, V]) {
	oldFreq := val.freq
	val.freq++

	// Remove from old freq list
	c.freqs[oldFreq].Remove(ent)
	if c.freqs[oldFreq].Len() == 0 {
		delete(c.freqs, oldFreq)
		if c.minFreq == oldFreq {
			c.minFreq++
		}
	}

	// Add to new freq list
	if c.freqs[val.freq] == nil {
		c.freqs[val.freq] = list.New()
	}
	newEnt := c.freqs[val.freq].PushFront(val)
	c.items[val.key] = newEnt
}

func (c *Cache[K, V]) evict() {
	l := c.freqs[c.minFreq]
	if l == nil {
		return // Should not happen
	}

	ent := l.Back()
	if ent != nil {
		l.Remove(ent)
		val := ent.Value.(*entry[K, V])
		delete(c.items, val.key)

		if l.Len() == 0 {
			delete(c.freqs, c.minFreq)
		}
	}
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
