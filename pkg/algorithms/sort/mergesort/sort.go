package mergesort

import "golang.org/x/exp/constraints"

// Sort sorts the slice s in ascending order using Merge Sort.
// Time Complexity: O(n log n). Stable.
func Sort[T constraints.Ordered](s []T) []T {
	if len(s) <= 1 {
		return s
	}

	mid := len(s) / 2
	left := Sort(s[:mid])
	right := Sort(s[mid:])

	return merge(left, right)
}

func merge[T constraints.Ordered](left, right []T) []T {
	result := make([]T, 0, len(left)+len(right))
	i, j := 0, 0

	for i < len(left) && j < len(right) {
		if left[i] <= right[j] { // Stable check
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
	}

	result = append(result, left[i:]...)
	result = append(result, right[j:]...)

	return result
}
