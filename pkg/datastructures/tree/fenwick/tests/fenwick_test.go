package fenwick_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/tree/fenwick"
	"testing"
)

func TestFenwickTree(t *testing.T) {
	ft := fenwick.New(10)

	ft.Add(1, 5)
	ft.Add(3, 2)

	if sum := ft.Query(5); sum != 7 {
		t.Errorf("Expected sum 7, got %d", sum)
	}

	if sum := ft.Query(2); sum != 5 {
		t.Errorf("Expected sum 5, got %d", sum)
	}
}
