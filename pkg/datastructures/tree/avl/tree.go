package avl

import (
	"sync"

	"golang.org/x/exp/constraints"
)

// Tree is an AVL Tree (self-balancing BST).
type Tree[K constraints.Ordered, V any] struct {
	root *node[K, V]
	mu   sync.RWMutex
}

type node[K constraints.Ordered, V any] struct {
	key    K
	value  V
	height int
	left   *node[K, V]
	right  *node[K, V]
}

func New[K constraints.Ordered, V any]() *Tree[K, V] {
	return &Tree[K, V]{}
}

func (t *Tree[K, V]) Put(key K, value V) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.root = insert(t.root, key, value)
}

func (t *Tree[K, V]) Get(key K) (V, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n := search(t.root, key)
	if n == nil {
		var zero V
		return zero, false
	}
	return n.value, true
}

func height[K constraints.Ordered, V any](n *node[K, V]) int {
	if n == nil {
		return 0
	}
	return n.height
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func updateHeight[K constraints.Ordered, V any](n *node[K, V]) {
	n.height = 1 + max(height(n.left), height(n.right))
}

func getBalance[K constraints.Ordered, V any](n *node[K, V]) int {
	if n == nil {
		return 0
	}
	return height(n.left) - height(n.right)
}

func rightRotate[K constraints.Ordered, V any](y *node[K, V]) *node[K, V] {
	x := y.left
	T2 := x.right

	x.right = y
	y.left = T2

	updateHeight(y)
	updateHeight(x)

	return x
}

func leftRotate[K constraints.Ordered, V any](x *node[K, V]) *node[K, V] {
	y := x.right
	T2 := y.left

	y.left = x
	x.right = T2

	updateHeight(x)
	updateHeight(y)

	return y
}

func insert[K constraints.Ordered, V any](n *node[K, V], key K, value V) *node[K, V] {
	if n == nil {
		return &node[K, V]{key: key, value: value, height: 1}
	}

	if key < n.key {
		n.left = insert(n.left, key, value)
	} else if key > n.key {
		n.right = insert(n.right, key, value)
	} else {
		n.value = value // Update value
		return n
	}

	updateHeight(n)
	balance := getBalance(n)

	// Left Left
	if balance > 1 && key < n.left.key {
		return rightRotate(n)
	}
	// Right Right
	if balance < -1 && key > n.right.key {
		return leftRotate(n)
	}
	// Left Right
	if balance > 1 && key > n.left.key {
		n.left = leftRotate(n.left)
		return rightRotate(n)
	}
	// Right Left
	if balance < -1 && key < n.right.key {
		n.right = rightRotate(n.right)
		return leftRotate(n)
	}

	return n
}

func search[K constraints.Ordered, V any](n *node[K, V], key K) *node[K, V] {
	if n == nil {
		return nil
	}
	if key < n.key {
		return search(n.left, key)
	} else if key > n.key {
		return search(n.right, key)
	}
	return n
}
