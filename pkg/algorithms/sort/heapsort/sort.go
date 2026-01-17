package heapsort

import "golang.org/x/exp/constraints"

// Sort sorts the slice s in ascending order using Heap Sort.
// Time Complexity: O(n log n). Not stable. In-place.
func Sort[T constraints.Ordered](s []T) {
	n := len(s)

	// Build Max Heap
	for i := n/2 - 1; i >= 0; i-- {
		heapify(s, n, i)
	}

	// Extract elements
	for i := n - 1; i > 0; i-- {
		s[0], s[i] = s[i], s[0] // Move root to end
		heapify(s, i, 0)        // Heapify reduced heap
	}
}

func heapify[T constraints.Ordered](s []T, n int, i int) {
	largest := i
	l := 2*i + 1
	r := 2*i + 2

	if l < n && s[l] > s[largest] {
		largest = l
	}

	if r < n && s[r] > s[largest] {
		largest = r
	}

	if largest != i {
		s[i], s[largest] = s[largest], s[i]
		heapify(s, n, largest)
	}
}
