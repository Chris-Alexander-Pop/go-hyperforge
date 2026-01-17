package bplus

import (
	"sort"
	"sync"

	"golang.org/x/exp/constraints"
)

const (
	degree = 4 // Minimum degree t. Node has [t-1, 2t-1] keys.
)

type Tree[K constraints.Ordered, V any] struct {
	root *node[K, V]
	mu   sync.RWMutex
}

type node[K constraints.Ordered, V any] struct {
	keys     []K
	children []*node[K, V]
	values   []V // Only for leaves
	isLeaf   bool
	next     *node[K, V] // Linked list for leaves
}

func New[K constraints.Ordered, V any]() *Tree[K, V] {
	return &Tree[K, V]{
		root: &node[K, V]{isLeaf: true},
	}
}

// Search returns the value for key.
func (t *Tree[K, V]) Search(key K) (V, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	n := t.root
	for !n.isLeaf {
		idx := 0
		found := false
		for i, k := range n.keys {
			if key < k {
				idx = i
				found = true
				break
			}
		}
		if !found {
			idx = len(n.keys)
		}
		n = n.children[idx]
	}

	// Leaf Search
	idx := sort.Search(len(n.keys), func(i int) bool { return n.keys[i] >= key })
	if idx < len(n.keys) && n.keys[idx] == key {
		return n.values[idx], true
	}
	var zero V
	return zero, false
}

// Insert adds a key-value pair.
func (t *Tree[K, V]) Insert(key K, value V) {
	t.mu.Lock()
	defer t.mu.Unlock()

	root := t.root
	if len(root.keys) == 2*degree-1 {
		newRoot := &node[K, V]{
			children: []*node[K, V]{root},
			isLeaf:   false,
		}
		t.splitChild(newRoot, 0)
		t.root = newRoot
		t.insertNonFull(newRoot, key, value)
	} else {
		t.insertNonFull(root, key, value)
	}
}

func (t *Tree[K, V]) insertNonFull(n *node[K, V], key K, value V) {
	if n.isLeaf {
		// Find position
		idx := sort.Search(len(n.keys), func(i int) bool { return n.keys[i] >= key })

		// Insert
		n.keys = append(n.keys, *new(K))
		copy(n.keys[idx+1:], n.keys[idx:])
		n.keys[idx] = key

		n.values = append(n.values, *new(V))
		copy(n.values[idx+1:], n.values[idx:])
		n.values[idx] = value
	} else {
		idx := 0
		found := false
		for i, k := range n.keys {
			if key < k {
				idx = i
				found = true
				break
			}
		}
		if !found {
			idx = len(n.keys)
		}

		child := n.children[idx]
		if len(child.keys) == 2*degree-1 {
			t.splitChild(n, idx)
			if key >= n.keys[idx] { // Key promoted
				idx++
			}
		}
		t.insertNonFull(n.children[idx], key, value)
	}
}

func (t *Tree[K, V]) splitChild(parent *node[K, V], index int) {
	child := parent.children[index]
	mid := degree - 1

	newChild := &node[K, V]{
		isLeaf: child.isLeaf,
	}

	if child.isLeaf {
		newChild.keys = append(newChild.keys, child.keys[mid:]...)
		newChild.values = append(newChild.values, child.values[mid:]...)

		child.keys = child.keys[:mid]
		child.values = child.values[:mid]

		newChild.next = child.next
		child.next = newChild

		promotedKey := newChild.keys[0]

		parent.keys = append(parent.keys, *new(K))
		copy(parent.keys[index+1:], parent.keys[index:])
		parent.keys[index] = promotedKey

		parent.children = append(parent.children, nil)
		copy(parent.children[index+2:], parent.children[index+1:])
		parent.children[index+1] = newChild

	} else {
		newChild.keys = append(newChild.keys, child.keys[mid+1:]...)
		newChild.children = append(newChild.children, child.children[mid+1:]...)

		promotedKey := child.keys[mid]

		child.keys = child.keys[:mid]
		child.children = child.children[:mid+1]

		parent.keys = append(parent.keys, *new(K))
		copy(parent.keys[index+1:], parent.keys[index:])
		parent.keys[index] = promotedKey

		parent.children = append(parent.children, nil)
		copy(parent.children[index+2:], parent.children[index+1:])
		parent.children[index+1] = newChild
	}
}
