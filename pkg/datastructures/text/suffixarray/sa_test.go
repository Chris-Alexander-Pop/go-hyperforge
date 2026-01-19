package suffixarray

import (
	"reflect"
	"sort"
	"testing"
)

func TestSuffixArray(t *testing.T) {
	sa := New("banana")

	// Test Search
	matches := sa.Search("ana")
	sort.Ints(matches)

	expected := []int{1, 3} // banana: indices 1, 3 start with 'ana'
	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("Expected %v, got %v", expected, matches)
	}

	// Test no match
	matches = sa.Search("xyz")
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matches))
	}
}
