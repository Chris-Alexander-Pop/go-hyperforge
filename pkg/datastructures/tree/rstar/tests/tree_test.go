package rstar_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/tree/rstar"
)

import "testing"

func TestRTree(t *testing.T) {
	rt := rstar.New()

	item1 := rstar.Item{Rect: rstar.Rect{MinX: 0, MinY: 0, MaxX: 10, MaxY: 10}, Data: "1"}
	item2 := rstar.Item{Rect: rstar.Rect{MinX: 20, MinY: 20, MaxX: 30, MaxY: 30}, Data: "2"}

	rt.Insert(item1)
	rt.Insert(item2)

	// Search intersecting item1
	found := rt.Search(rstar.Rect{MinX: 1, MinY: 1, MaxX: 5, MaxY: 5})
	if len(found) != 1 {
		t.Errorf("Expected 1 match, got %d", len(found))
	}

	// Search intersecting None
	found = rt.Search(rstar.Rect{MinX: 100, MinY: 100, MaxX: 110, MaxY: 110})
	if len(found) != 0 {
		t.Error("Expected 0 matches")
	}
}
