package disjoint

import "testing"

func TestDisjointSet(t *testing.T) {
	ds := New()

	ds.MakeSet("a")
	ds.MakeSet("b")
	ds.MakeSet("c")
	ds.MakeSet("d")

	if ds.Connected("a", "b") {
		t.Error("a and b should not be connected")
	}

	ds.Union("a", "b")
	if !ds.Connected("a", "b") {
		t.Error("a and b should be connected")
	}

	ds.Union("c", "d")
	ds.Union("a", "c")

	if !ds.Connected("b", "d") {
		t.Error("b and d should be connected transitively")
	}

	ds.MakeSet("b") // Should be no-op or safe
	if !ds.Connected("a", "b") {
		t.Error("MakeSet on existing element broke connectivity")
	}
}
