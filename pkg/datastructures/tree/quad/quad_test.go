package quad

import "testing"

func TestQuadtree(t *testing.T) {
	qt := New(Bounds{0, 0, 100, 100}, 4)

	p1 := Point{10, 10, "p1"}
	p2 := Point{90, 90, "p2"}

	qt.Insert(p1)
	qt.Insert(p2)

	// Query hitting p1
	found := qt.Query(Bounds{0, 0, 50, 50})
	if len(found) != 1 {
		t.Errorf("Expected 1 point, got %d", len(found))
	} else if found[0] != p1 {
		t.Error("Wrong point found")
	}

	// Query hitting p2
	found2 := qt.Query(Bounds{50, 50, 50, 50})
	if len(found2) != 1 {
		t.Errorf("Expected 1 point, got %d", len(found2))
	}
}
