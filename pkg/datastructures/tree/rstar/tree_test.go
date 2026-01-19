package rstar

import "testing"

func TestRTree(t *testing.T) {
	rt := New()

	item1 := Item{Rect: Rect{0, 0, 10, 10}, Data: "1"}
	item2 := Item{Rect: Rect{20, 20, 30, 30}, Data: "2"}

	rt.Insert(item1)
	rt.Insert(item2)

	// Search intersecting item1
	found := rt.Search(Rect{1, 1, 5, 5})
	if len(found) != 1 {
		t.Errorf("Expected 1 match, got %d", len(found))
	}

	// Search intersecting None
	found = rt.Search(Rect{100, 100, 110, 110})
	if len(found) != 0 {
		t.Error("Expected 0 matches")
	}
}
