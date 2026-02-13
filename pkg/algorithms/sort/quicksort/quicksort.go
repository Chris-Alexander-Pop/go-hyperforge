package quicksort

import (
	"golang.org/x/exp/constraints"
)

// Sort sorts the slice s in ascending order using Quick Sort.
// Time Complexity: Average O(n log n), Worst O(n^2). Not stable.
func Sort[T constraints.Ordered](s []T) {
	if len(s) <= 1 {
		return
	}

	// Median-of-three pivot selection
	// Reduces the chance of worst-case performance on sorted arrays
	// and eliminates the global lock contention of math/rand.Intn
	mid := len(s) / 2
	low, high := 0, len(s)-1

	if s[mid] < s[low] {
		s[low], s[mid] = s[mid], s[low]
	}
	if s[high] < s[low] {
		s[low], s[high] = s[high], s[low]
	}
	if s[high] < s[mid] {
		s[mid], s[high] = s[high], s[mid]
	}

	// s[mid] is now the median of s[low], s[mid], s[high].
	// Swap it with s[high] to use it as the pivot.
	s[mid], s[high] = s[high], s[mid]

	p := partition(s)

	Sort(s[:p])
	Sort(s[p+1:])
}

func partition[T constraints.Ordered](s []T) int {
	pivot := s[len(s)-1]
	i := -1

	for j := 0; j < len(s)-1; j++ {
		if s[j] < pivot {
			i++
			s[i], s[j] = s[j], s[i]
		}
	}

	s[i+1], s[len(s)-1] = s[len(s)-1], s[i+1]
	return i + 1
}
