package concurrency

import (
	"hash/crc32"
	"sort"
	"sync"
)

// HashRing implements consistent hashing with virtual nodes.
//
// Consistent hashing ensures that when nodes are added or removed,
// only a minimal number of keys need to be remapped.
//
// Virtual nodes improve distribution by mapping each physical node
// to multiple points on the ring.
type HashRing struct {
	nodes        map[string]struct{} // Physical nodes
	ring         []uint32            // Sorted hash values
	hashToNode   map[uint32]string   // Hash -> node mapping
	virtualNodes int                 // Virtual nodes per physical node
	mu           sync.RWMutex
}

// NewHashRing creates a new consistent hash ring.
// virtualNodes controls the number of virtual nodes per physical node.
// Higher values provide better distribution but use more memory.
// Recommended: 100-200 for most use cases.
func NewHashRing(virtualNodes int) *HashRing {
	if virtualNodes <= 0 {
		virtualNodes = 150
	}
	return &HashRing{
		nodes:        make(map[string]struct{}),
		ring:         make([]uint32, 0),
		hashToNode:   make(map[uint32]string),
		virtualNodes: virtualNodes,
	}
}

// AddNode adds a node to the ring.
func (h *HashRing) AddNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.nodes[node]; exists {
		return
	}

	h.nodes[node] = struct{}{}

	// Add virtual nodes
	for i := 0; i < h.virtualNodes; i++ {
		hash := h.hashKey(virtualNodeKey(node, i))
		h.ring = append(h.ring, hash)
		h.hashToNode[hash] = node
	}

	// Sort ring
	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
}

// RemoveNode removes a node from the ring.
func (h *HashRing) RemoveNode(node string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.nodes[node]; !exists {
		return
	}

	delete(h.nodes, node)

	// Remove virtual nodes
	for i := 0; i < h.virtualNodes; i++ {
		hash := h.hashKey(virtualNodeKey(node, i))
		delete(h.hashToNode, hash)
	}

	// Rebuild ring without removed node's hashes
	newRing := make([]uint32, 0, len(h.ring)-h.virtualNodes)
	for _, hash := range h.ring {
		if _, exists := h.hashToNode[hash]; exists {
			newRing = append(newRing, hash)
		}
	}
	h.ring = newRing
}

// GetNode returns the node responsible for the given key.
func (h *HashRing) GetNode(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return ""
	}

	hash := h.hashKey(key)

	// Binary search for the first node with hash >= key hash
	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})

	// Wrap around to the beginning if needed
	if idx >= len(h.ring) {
		idx = 0
	}

	return h.hashToNode[h.ring[idx]]
}

// GetNodes returns n nodes responsible for the given key (for replication).
func (h *HashRing) GetNodes(key string, n int) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return nil
	}

	nodeCount := len(h.nodes)
	if n > nodeCount {
		n = nodeCount
	}

	hash := h.hashKey(key)

	idx := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= hash
	})

	if idx >= len(h.ring) {
		idx = 0
	}

	seen := make(map[string]struct{})
	result := make([]string, 0, n)

	for len(result) < n {
		node := h.hashToNode[h.ring[idx]]
		if _, exists := seen[node]; !exists {
			seen[node] = struct{}{}
			result = append(result, node)
		}
		idx = (idx + 1) % len(h.ring)
	}

	return result
}

// Nodes returns all physical nodes in the ring.
func (h *HashRing) Nodes() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	nodes := make([]string, 0, len(h.nodes))
	for node := range h.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Size returns the number of physical nodes.
func (h *HashRing) Size() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.nodes)
}

func (h *HashRing) hashKey(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}

func virtualNodeKey(node string, index int) string {
	return node + "#" + string(rune(index))
}
