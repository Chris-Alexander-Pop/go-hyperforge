package fenwick

// Tree (Binary Indexed Tree) provides efficient prefix sums and updates.
type Tree struct {
	tree []int
}

// New creates a new Fenwick Tree with the given size.
func New(size int) *Tree {
	return &Tree{
		tree: make([]int, size+1),
	}
}

// Add adds delta to the element at index i (1-based).
func (t *Tree) Add(i int, delta int) {
	for i < len(t.tree) {
		t.tree[i] += delta
		i += i & -i
	}
}

// Query returns the prefix sum up to index i (inclusive).
func (t *Tree) Query(i int) int {
	sum := 0
	for i > 0 {
		sum += t.tree[i]
		i -= i & -i
	}
	return sum
}

// RangeQuery returns sum from i to j (inclusive).
func (t *Tree) RangeQuery(i, j int) int {
	return t.Query(j) - t.Query(i-1)
}
