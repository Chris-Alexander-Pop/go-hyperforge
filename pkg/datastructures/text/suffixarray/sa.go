package suffixarray

import (
	"sort"
)

// SuffixArray provides efficient substring search.
type SuffixArray struct {
	data    string
	indices []int
}

// New creates a Suffix Array for the given text.
// Uses simple O(n^2 log n) sort construction for readability.
// For production, use O(n) SA-IS or skew algorithms.
func New(text string) *SuffixArray {
	n := len(text)
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}

	sort.Slice(indices, func(i, j int) bool {
		return text[indices[i]:] < text[indices[j]:]
	})

	return &SuffixArray{
		data:    text,
		indices: indices,
	}
}

// Search returns all start indices of the substring.
func (sa *SuffixArray) Search(sub string) []int {
	n := len(sa.data)
	// Binary search for lower bound
	l, r := 0, n
	for l < r {
		mid := (l + r) / 2
		suffix := sa.data[sa.indices[mid]:]
		if suffix < sub {
			l = mid + 1
		} else {
			r = mid
		}
	}
	start := l

	// Binary search for upper bound logic (omitted, simple iteration for range)
	// Iterate to collect matches
	var matches []int
	for i := start; i < n; i++ {
		suffix := sa.data[sa.indices[i]:]
		if len(suffix) < len(sub) || suffix[:len(sub)] != sub {
			break
		}
		matches = append(matches, sa.indices[i])
	}
	return matches
}
