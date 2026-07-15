package binarysearch

import "cmp"

// Search returns the index of target in ascending-sorted s.
// If target is not present, found is false and index is the insertion point
// (the smallest index where target could be inserted while preserving order).
func Search[T cmp.Ordered](s []T, target T) (index int, found bool) {
	lo, hi := 0, len(s)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if s[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(s) && s[lo] == target {
		return lo, true
	}
	return lo, false
}

// SearchFunc searches for the smallest index i in [0, len(s)) such that
// ok(i) is true, assuming ok is false for all indices below some point and
// true thereafter (the same contract as sort.Search).
// If there is no such index, it returns len(s).
func SearchFunc(n int, ok func(i int) bool) int {
	lo, hi := 0, n
	for lo < hi {
		mid := lo + (hi-lo)/2
		if !ok(mid) {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// LowerBound returns the first index i where s[i] >= target.
// If all elements are less than target, returns len(s).
func LowerBound[T cmp.Ordered](s []T, target T) int {
	return SearchFunc(len(s), func(i int) bool { return s[i] >= target })
}

// UpperBound returns the first index i where s[i] > target.
// If all elements are less than or equal to target, returns len(s).
func UpperBound[T cmp.Ordered](s []T, target T) int {
	return SearchFunc(len(s), func(i int) bool { return s[i] > target })
}
