package mergesort_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/sort/mergesort"
	"reflect"
	"testing"
)

func TestSort(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		want  []int
	}{
		{"empty", []int{}, []int{}},
		{"single", []int{1}, []int{1}},
		{"sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"reverse", []int{3, 2, 1}, []int{1, 2, 3}},
		{"duplicates", []int{1, 2, 2, 1}, []int{1, 1, 2, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergesort.Sort(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergesort.Sort() = %v, want %v", got, tt.want)
			}
		})
	}
}
