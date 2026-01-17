package radix

import (
	"strings"
	"sync"
)

// RadixTree is a compressed prefix tree.
// Great for routing (fewer nodes than Trie).
type RadixTree[V any] struct {
	root *node[V]
	mu   sync.RWMutex
}

type node[V any] struct {
	path     string
	indices  string // first char of edges (optimization)
	children []*node[V]
	value    V
	isTerm   bool
}

func New[V any]() *RadixTree[V] {
	return &RadixTree[V]{
		root: &node[V]{},
	}
}

// Insert adds a path.
func (t *RadixTree[V]) Insert(path string, value V) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// find longest common prefix
	// ... logic was here but messy.
	// We delegate all logic to the recursive insert function which handles
	// root cases, edge splitting, and node creation properly.
	t.insert(t.root, path, value)
}

// insert helper recursive (re-implementation of logic)
func (t *RadixTree[V]) insert(n *node[V], path string, value V) {
	// 1. At root or current node, look for child starting with path[0]
	if len(path) == 0 {
		n.isTerm = true
		n.value = value
		return
	}

	for _, child := range n.children {
		if child.path[0] == path[0] {
			// Found matching edge
			// Determine shared prefix length
			lcp := 0
			minLen := len(child.path)
			if len(path) < minLen {
				minLen = len(path)
			}
			for lcp < minLen && child.path[lcp] == path[lcp] {
				lcp++
			}

			if lcp == len(child.path) {
				// Full match on edge, go deeper
				t.insert(child, path[lcp:], value)
				return
			}

			// Partial match - Split the edge
			// Current child becomes:
			//   Head (prefix match)
			//     -> Tail (remaining old path)
			//     -> New Node (remaining new path)

			// Create tail node (preserves old child's data)
			tail := &node[V]{
				path:     child.path[lcp:],
				value:    child.value,
				isTerm:   child.isTerm,
				children: child.children,
			}

			// Create new leaf node for new value
			var leaf *node[V]
			if lcp < len(path) {
				leaf = &node[V]{
					path:   path[lcp:],
					value:  value,
					isTerm: true,
				}
			} else {
				// The insertion ends exactly at the split point
				// The 'head' logic below handles setting value on the split node
			}

			// Update existing child to be the 'head' (split node)
			child.path = child.path[:lcp]
			child.children = []*node[V]{tail}
			if leaf != nil {
				child.children = append(child.children, leaf)
				child.isTerm = false // It was just a path segment, unless...
			} else {
				// Split point IS the new value
				child.value = value
				child.isTerm = true
			}

			// Fix up n.children[i] is effectively updated since child is a pointer
			return
		}
	}

	// No match found, add new child
	child := &node[V]{
		path:   path,
		value:  value,
		isTerm: true,
	}
	n.children = append(n.children, child)
}

// Get retrieves a value.
func (t *RadixTree[V]) Get(path string) (V, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.get(t.root, path)
}

func (t *RadixTree[V]) get(n *node[V], path string) (V, bool) {
	if len(path) == 0 {
		if n.isTerm {
			return n.value, true
		}
		var zero V
		return zero, false
	}

	for _, child := range n.children {
		if strings.HasPrefix(path, child.path) {
			return t.get(child, path[len(child.path):])
		}
	}

	var zero V
	return zero, false
}
