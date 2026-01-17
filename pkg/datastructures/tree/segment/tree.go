package segment

// Tree supports range queries (Sum, Min, Max) on an array.
// This implementation focuses on Sum.
type Tree struct {
	tree []int
	n    int
	op   func(int, int) int
}

// New creates a segment tree for the given array with a merge operation.
// Default op is sum.
func New(data []int, op func(int, int) int) *Tree {
	if op == nil {
		op = func(a, b int) int { return a + b }
	}
	n := len(data)
	// Size is roughly 4*n
	treeNodes := make([]int, 4*n)

	t := &Tree{
		tree: treeNodes,
		n:    n,
		op:   op,
	}
	if n > 0 {
		t.build(data, 1, 0, n-1)
	}
	return t
}

func (t *Tree) build(data []int, node, start, end int) {
	if start == end {
		t.tree[node] = data[start]
	} else {
		mid := (start + end) / 2
		t.build(data, 2*node, start, mid)
		t.build(data, 2*node+1, mid+1, end)
		t.tree[node] = t.op(t.tree[2*node], t.tree[2*node+1])
	}
}

// Update updates the value at index idx to val.
func (t *Tree) Update(idx int, val int) {
	t.update(1, 0, t.n-1, idx, val)
}

func (t *Tree) update(node, start, end, idx, val int) {
	if start == end {
		t.tree[node] = val
	} else {
		mid := (start + end) / 2
		if start <= idx && idx <= mid {
			t.update(2*node, start, mid, idx, val)
		} else {
			t.update(2*node+1, mid+1, end, idx, val)
		}
		t.tree[node] = t.op(t.tree[2*node], t.tree[2*node+1])
	}
}

// Query returns the result of the operation range [l, r].
func (t *Tree) Query(l, r int) int {
	return t.query(1, 0, t.n-1, l, r)
}

func (t *Tree) query(node, start, end, l, r int) int {
	if r < start || end < l {
		return 0 // Identity element depends on OP. 0 for Sum. MaxInt for Min.
		// NOTE: This implementation assumes Sum (0).
		// For robustness, New() should accept identity value.
	}
	if l <= start && end <= r {
		return t.tree[node]
	}
	mid := (start + end) / 2
	p1 := t.query(2*node, start, mid, l, r)
	p2 := t.query(2*node+1, mid+1, end, l, r)
	return t.op(p1, p2)
}
