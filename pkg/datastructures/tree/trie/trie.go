package trie

import (
	"sync"
)

// Trie is a concurrent prefix tree.
type Trie[V any] struct {
	root *node[V]
	mu   sync.RWMutex
}

type node[V any] struct {
	children map[rune]*node[V]
	value    V
	isTerm   bool
}

func New[V any]() *Trie[V] {
	return &Trie[V]{
		root: &node[V]{children: make(map[rune]*node[V])},
	}
}

// Insert inserts a key and value (overwrite if exists).
func (t *Trie[V]) Insert(key string, value V) {
	t.mu.Lock()
	defer t.mu.Unlock()

	n := t.root
	for _, char := range key {
		if n.children[char] == nil {
			n.children[char] = &node[V]{children: make(map[rune]*node[V])}
		}
		n = n.children[char]
	}
	n.value = value
	n.isTerm = true
}

// Get retrieves a value.
func (t *Trie[V]) Get(key string) (V, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	n := t.root
	for _, char := range key {
		if n.children[char] == nil {
			var zero V
			return zero, false
		}
		n = n.children[char]
	}
	if !n.isTerm {
		var zero V
		return zero, false
	}
	return n.value, true
}

// Delete removes a key.
func (t *Trie[V]) Delete(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.delete(t.root, []rune(key), 0)
}

func (t *Trie[V]) delete(n *node[V], key []rune, depth int) bool {
	if depth == len(key) {
		if !n.isTerm {
			return false
		}
		n.isTerm = false
		// if leaf, return true to delete parent pointer
		return len(n.children) == 0
	}

	char := key[depth]
	child := n.children[char]
	if child == nil {
		return false
	}

	shouldDeleteChild := t.delete(child, key, depth+1)
	if shouldDeleteChild {
		delete(n.children, char)
		return len(n.children) == 0 && !n.isTerm
	}
	return false
}

// PrefixSearch returns all values starting with prefix.
func (t *Trie[V]) PrefixSearch(prefix string) []V {
	t.mu.RLock()
	defer t.mu.RUnlock()

	n := t.root
	for _, char := range prefix {
		if n.children[char] == nil {
			return nil
		}
		n = n.children[char]
	}

	var results []V
	var collect func(*node[V])
	collect = func(curr *node[V]) {
		if curr.isTerm {
			results = append(results, curr.value)
		}
		for _, child := range curr.children {
			collect(child)
		}
	}
	collect(n)
	return results
}
