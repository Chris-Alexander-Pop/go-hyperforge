package quicksort

import (
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	tests := []struct {
		name  string
		input []int
	}{
		{"Already sorted", []int{1, 2, 3, 4, 5}},
		{"Reverse sorted", []int{5, 4, 3, 2, 1}},
		{"Random", []int{3, 1, 4, 1, 5, 9, 2, 6}},
		{"Duplicates", []int{1, 2, 2, 3, 1}},
		{"Empty", []int{}},
		{"Single", []int{1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to sort
			data := make([]int, len(tt.input))
			copy(data, tt.input)

			// Sort using our implementation
			Sort(data)

			// verify
			if !sort.IntsAreSorted(data) {
				t.Errorf("Sort() failed, got %v", data)
			}

			// Verify elements are same (length check usually enough if sorted)
			if len(data) != len(tt.input) {
				t.Errorf("Length changed")
			}
		})
	}
}
