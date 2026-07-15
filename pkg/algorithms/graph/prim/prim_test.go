package prim_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/graph/prim"
)

func TestMST(t *testing.T) {
	g := prim.Graph{
		"A": {"B": 1, "C": 4},
		"B": {"A": 1, "C": 2, "D": 5},
		"C": {"A": 4, "B": 2, "D": 1},
		"D": {"B": 5, "C": 1},
	}
	edges, w := prim.MST(g)
	if len(edges) != 3 {
		t.Fatalf("edges=%d want 3: %+v", len(edges), edges)
	}
	if w != 4 { // A-B(1) + B-C(2) + C-D(1)
		t.Fatalf("weight=%v want 4", w)
	}
}
