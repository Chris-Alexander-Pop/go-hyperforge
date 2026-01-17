package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Ring implements a consistent hashing ring.
type Ring struct {
	replicas int            // Number of virtual nodes per physical node
	keys     []int          // Sorted hash values
	hashMap  map[int]string // Map of hash key to physical node
	mu       *concurrency.SmartRWMutex
}

// New creates a new consistent hash ring.
// replicas: Number of virtual nodes per physical node (e.g., 50-200).
// nodes: Optional list of initial physical nodes.
func New(replicas int, nodes []string) *Ring {
	if replicas <= 0 {
		replicas = 50
	}
	r := &Ring{
		replicas: replicas,
		hashMap:  make(map[int]string),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "ConsistentHashRing"}),
	}
	for _, node := range nodes {
		r.Add(node)
	}
	return r
}

// Add adds a physical node to the ring.
func (r *Ring) Add(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < r.replicas; i++ {
		hash := int(crc32.ChecksumIEEE([]byte(strconv.Itoa(i) + node)))
		r.keys = append(r.keys, hash)
		r.hashMap[hash] = node
	}
	sort.Ints(r.keys)
}

// Remove removes a physical node from the ring.
func (r *Ring) Remove(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// This is O(N*replicas), which can be slow for large rings
	// A more optimized approach would track which hashes belong to which node
	newKeys := make([]int, 0, len(r.keys))
	for _, k := range r.keys {
		if r.hashMap[k] != node {
			newKeys = append(newKeys, k)
		} else {
			delete(r.hashMap, k)
		}
	}
	r.keys = newKeys
}

// Get returns the closest node for the given key.
func (r *Ring) Get(key string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		return ""
	}

	hash := int(crc32.ChecksumIEEE([]byte(key)))

	// Binary search for first item >= hash
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	// Wrap around
	if idx == len(r.keys) {
		idx = 0
	}

	return r.hashMap[r.keys[idx]]
}

// GetN returns the N distinct nodes responsible for the given key (for replication).
func (r *Ring) GetN(key string, n int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.keys) == 0 {
		return nil
	}

	hash := int(crc32.ChecksumIEEE([]byte(key)))
	idx := sort.Search(len(r.keys), func(i int) bool {
		return r.keys[i] >= hash
	})

	if idx == len(r.keys) {
		idx = 0
	}

	seen := make(map[string]bool)
	result := make([]string, 0, n)

	// Walk the ring
	for len(result) < n && len(seen) < len(r.keys)/r.replicas {
		node := r.hashMap[r.keys[idx]]
		if !seen[node] {
			seen[node] = true
			result = append(result, node)
		}
		idx = (idx + 1) % len(r.keys)
	}

	return result
}
