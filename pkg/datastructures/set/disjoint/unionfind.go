package disjoint

import "sync"

// Set implements a Disjoint Set (Union-Find) with path compression and rank optimization.
// Thread-safe.
type Set struct {
	parent map[string]string
	rank   map[string]int
	mu     sync.RWMutex
}

func New() *Set {
	return &Set{
		parent: make(map[string]string),
		rank:   make(map[string]int),
	}
}

// MakeSet creates a new set with a single element.
func (s *Set) MakeSet(x string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.parent[x]; !exists {
		s.parent[x] = x
		s.rank[x] = 0
	}
}

// Find returns the representative of the set containing x.
func (s *Set) Find(x string) string {
	s.mu.Lock() // Lock needed for path compression
	defer s.mu.Unlock()
	return s.find(x)
}

func (s *Set) find(x string) string {
	if _, exists := s.parent[x]; !exists {
		return ""
	}
	if s.parent[x] != x {
		s.parent[x] = s.find(s.parent[x]) // Path compression
	}
	return s.parent[x]
}

// Union merges the sets containing x and y.
func (s *Set) Union(x, y string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rootX := s.find(x)
	rootY := s.find(y)

	if rootX == "" || rootY == "" || rootX == rootY {
		return
	}

	// Union by rank
	if s.rank[rootX] < s.rank[rootY] {
		s.parent[rootX] = rootY
	} else if s.rank[rootX] > s.rank[rootY] {
		s.parent[rootY] = rootX
	} else {
		s.parent[rootY] = rootX
		s.rank[rootX]++
	}
}

// Connected checks if x and y are in the same set.
func (s *Set) Connected(x, y string) bool {
	// Find acquires lock, so standard public method is fine if we accept
	// slightly loose concurrency (x could move sets between calls).
	// But Find() locks, so it's safe individually.
	// For strict atomicity, lock inside.
	s.mu.RLock() // RLock ok if we don't compress? No, Find needs write lock for compression.
	// We'll use full lock or expose internal non-compressing find?
	// Standard Find compresses, which is mutation.
	s.mu.RUnlock()

	return s.Find(x) == s.Find(y)
}
