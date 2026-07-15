package arc

import (
	"container/list"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Cache is a thread-safe Adaptive Replacement Cache.
type Cache[K comparable, V any] struct {
	mu       *concurrency.SmartRWMutex
	capacity int
	p        int        // target size for t1
	t1, t2   *list.List // active lists (MRU positions of L1/L2)
	b1, b2   *list.List // ghost lists (LRU positions of L1/L2)
	items    map[K]*entry[K, V]
}

type entry[K comparable, V any] struct {
	key     K
	value   V
	isGhost bool
	el      *list.Element
	listID  int // 0=none, 1=t1, 2=t2, 3=b1, 4=b2
}

func New[K comparable, V any](capacity int) *Cache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &Cache[K, V]{
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "ARC"}),
		capacity: capacity,
		p:        0,
		t1:       list.New(),
		t2:       list.New(),
		b1:       list.New(),
		b2:       list.New(),
		items:    make(map[K]*entry[K, V]),
	}
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent, exists := c.items[key]
	if !exists || ent.isGhost {
		var zero V
		return zero, false
	}

	// Cache hit in t1 or t2
	// Move to t2 (MRU)
	c.remove(ent)
	ent.listID = 2
	ent.el = c.t2.PushFront(ent)

	return ent.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ent, exists := c.items[key]

	// Case 1: x in T1 or T2 (Hit)
	if exists && !ent.isGhost {
		ent.value = value
		c.remove(ent)
		ent.listID = 2
		ent.el = c.t2.PushFront(ent)
		return
	}

	// Case 2: x in B1 (Miss, but in history B1)
	if exists && ent.listID == 3 {
		// Adapt p
		delta := 1
		if c.b1.Len() < c.b2.Len() {
			delta = c.b2.Len() / c.b1.Len()
		}
		c.p = min(c.p+delta, c.capacity)

		c.replace(ent)

		// Move to t2
		c.remove(ent)
		ent.isGhost = false
		ent.value = value
		ent.listID = 2
		ent.el = c.t2.PushFront(ent)
		return
	}

	// Case 3: x in B2 (Miss, but in history B2)
	if exists && ent.listID == 4 {
		// Adapt p
		delta := 1
		if c.b2.Len() < c.b1.Len() {
			delta = c.b1.Len() / c.b2.Len()
		}
		c.p = max(c.p-delta, 0)

		c.replace(ent)

		// Move to t2
		c.remove(ent)
		ent.isGhost = false
		ent.value = value
		ent.listID = 2
		ent.el = c.t2.PushFront(ent)
		return
	}

	// Case 4: x not in T1, T2, B1, B2 (Complete Miss)
	// If L1 (T1+B1) has capacity L1 or simple full check logic
	if c.t1.Len()+c.b1.Len() == c.capacity {
		if c.t1.Len() < c.capacity {
			c.evictLRU(c.b1)
			c.replace(nil)
		} else {
			c.evictLRU(c.t1) // del from t1, just drop? no, drop
		}
	} else if c.t1.Len()+c.b1.Len() < c.capacity {
		if c.t1.Len()+c.t2.Len()+c.b1.Len()+c.b2.Len() >= c.capacity {
			if c.t1.Len()+c.t2.Len()+c.b1.Len()+c.b2.Len() == 2*c.capacity {
				c.evictLRU(c.b2)
			}
			c.replace(nil)
		}
	}

	// Insert into T1
	newEnt := &entry[K, V]{key: key, value: value, listID: 1, isGhost: false}
	newEnt.el = c.t1.PushFront(newEnt)
	c.items[key] = newEnt
}

func (c *Cache[K, V]) replace(current *entry[K, V]) {
	if c.t1.Len() > 0 && (c.t1.Len() > c.p || (current != nil && current.listID == 4 && c.t1.Len() == c.p)) {
		// Move LRU of T1 to B1
		el := c.t1.Back()
		c.t1.Remove(el)
		ent := el.Value.(*entry[K, V])
		ent.isGhost = true
		ent.value = *new(V) // clear value to save memory
		ent.listID = 3
		ent.el = c.b1.PushFront(ent)
	} else {
		// Move LRU of T2 to B2
		el := c.t2.Back()
		if el != nil {
			c.t2.Remove(el)
			ent := el.Value.(*entry[K, V])
			ent.isGhost = true
			ent.value = *new(V)
			ent.listID = 4
			ent.el = c.b2.PushFront(ent)
		}
	}
}

func (c *Cache[K, V]) remove(ent *entry[K, V]) {
	switch ent.listID {
	case 1:
		c.t1.Remove(ent.el)
	case 2:
		c.t2.Remove(ent.el)
	case 3:
		c.b1.Remove(ent.el)
	case 4:
		c.b2.Remove(ent.el)
	}
}

func (c *Cache[K, V]) evictLRU(l *list.List) {
	if l.Len() == 0 {
		return
	}
	el := l.Back()
	l.Remove(el)
	ent := el.Value.(*entry[K, V])
	delete(c.items, ent.key)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
