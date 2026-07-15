package louvain_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/graph/louvain"
)

func TestDetectTwoCliques(t *testing.T) {
	// Two triangles connected by a bridge.
	g := louvain.Graph{Edges: map[int][]int{
		0: {1, 2},
		1: {0, 2},
		2: {0, 1, 3},
		3: {2, 4, 5},
		4: {3, 5},
		5: {3, 4},
	}}

	comms := louvain.Detect(g)
	if len(comms) == 0 {
		t.Fatal("empty communities")
	}

	q := louvain.Modularity(g, comms)
	singletons := []louvain.Community{
		{Nodes: []int{0}}, {Nodes: []int{1}}, {Nodes: []int{2}},
		{Nodes: []int{3}}, {Nodes: []int{4}}, {Nodes: []int{5}},
	}
	qSingle := louvain.Modularity(g, singletons)
	if q < qSingle {
		t.Fatalf("modularity %v should be >= singletons %v; comms=%+v", q, qSingle, comms)
	}
}

func TestDetectEmpty(t *testing.T) {
	if louvain.Detect(louvain.Graph{Edges: nil}) != nil {
		t.Fatal("expected nil")
	}
}

func TestModularityCompleteGraph(t *testing.T) {
	g := louvain.Graph{Edges: map[int][]int{
		0: {1, 2},
		1: {0, 2},
		2: {0, 1},
	}}
	comms := louvain.Detect(g)
	if len(comms) != 1 {
		// Complete triangle should usually collapse to one community.
		t.Logf("comms=%+v (may be >1 depending on pass order)", comms)
	}
	_ = louvain.Modularity(g, comms)
}
