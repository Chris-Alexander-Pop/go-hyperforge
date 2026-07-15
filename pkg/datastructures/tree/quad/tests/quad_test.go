package quad_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/tree/quad"
)

import "testing"

func TestQuadtree(t *testing.T) {
	qt := quad.New(quad.Bounds{X: 0, Y: 0, W: 100, H: 100}, 4)

	p1 := quad.Point{X: 10, Y: 10, Data: "p1"}
	p2 := quad.Point{X: 90, Y: 90, Data: "p2"}

	qt.Insert(p1)
	qt.Insert(p2)

	// Query hitting p1
	found := qt.Query(quad.Bounds{X: 0, Y: 0, W: 50, H: 50})
	if len(found) != 1 {
		t.Errorf("Expected 1 point, got %d", len(found))
	} else if found[0] != p1 {
		t.Error("Wrong point found")
	}

	// Query hitting p2
	found2 := qt.Query(quad.Bounds{X: 50, Y: 50, W: 50, H: 50})
	if len(found2) != 1 {
		t.Errorf("Expected 1 point, got %d", len(found2))
	}
}
