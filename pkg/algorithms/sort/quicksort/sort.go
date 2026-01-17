package quicksort

import (
	"math/rand"

	"golang.org/x/exp/constraints"
)

// Sort sorts the slice s in ascending order using Quick Sort.
// Time Complexity: Average O(n log n), Worst O(n^2). Not stable.
func Sort[T constraints.Ordered](s []T) {
	if len(s) <= 1 {
		return
	}

	// Randomized pivot for better average performance
	pivotIdx := rand.Intn(len(s))
	s[pivotIdx], s[len(s)-1] = s[len(s)-1], s[pivotIdx]

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
