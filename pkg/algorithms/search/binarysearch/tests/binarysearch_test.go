package binarysearch_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/search/binarysearch"
)

func TestSearch(t *testing.T) {
	s := []int{1, 3, 5, 7, 9}

	idx, found := binarysearch.Search(s, 5)
	if !found || idx != 2 {
		t.Fatalf("Search(5) = (%d, %v), want (2, true)", idx, found)
	}

	idx, found = binarysearch.Search(s, 4)
	if found || idx != 2 {
		t.Fatalf("Search(4) = (%d, %v), want (2, false)", idx, found)
	}

	idx, found = binarysearch.Search(s, 0)
	if found || idx != 0 {
		t.Fatalf("Search(0) = (%d, %v), want (0, false)", idx, found)
	}

	idx, found = binarysearch.Search(s, 10)
	if found || idx != 5 {
		t.Fatalf("Search(10) = (%d, %v), want (5, false)", idx, found)
	}

	idx, found = binarysearch.Search([]int{}, 1)
	if found || idx != 0 {
		t.Fatalf("Search(empty) = (%d, %v), want (0, false)", idx, found)
	}
}

func TestSearchStrings(t *testing.T) {
	s := []string{"a", "c", "e"}
	idx, found := binarysearch.Search(s, "c")
	if !found || idx != 1 {
		t.Fatalf("Search(\"c\") = (%d, %v), want (1, true)", idx, found)
	}
}

func TestSearchFunc(t *testing.T) {
	s := []int{1, 3, 5, 7, 9}
	i := binarysearch.SearchFunc(len(s), func(i int) bool { return s[i] >= 6 })
	if i != 3 {
		t.Fatalf("SearchFunc >= 6 = %d, want 3", i)
	}
	i = binarysearch.SearchFunc(len(s), func(i int) bool { return s[i] >= 10 })
	if i != 5 {
		t.Fatalf("SearchFunc >= 10 = %d, want 5", i)
	}
}

func TestLowerUpperBound(t *testing.T) {
	s := []int{1, 2, 2, 2, 4}
	if got := binarysearch.LowerBound(s, 2); got != 1 {
		t.Fatalf("LowerBound(2) = %d, want 1", got)
	}
	if got := binarysearch.UpperBound(s, 2); got != 4 {
		t.Fatalf("UpperBound(2) = %d, want 4", got)
	}
	if got := binarysearch.LowerBound(s, 3); got != 4 {
		t.Fatalf("LowerBound(3) = %d, want 4", got)
	}
}
