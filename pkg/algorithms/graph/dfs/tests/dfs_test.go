package dfs_test

import (
	"reflect"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph"
	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph/dfs"
)

func TestReachable(t *testing.T) {
	g := graph.AdjacencyList{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"E"},
		"D": {},
		"E": {},
	}
	got := dfs.Reachable(g, "A")
	// Stack pushes neighbors in reverse, so B is explored before C.
	want := []string{"A", "B", "D", "C", "E"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reachable = %v, want %v", got, want)
	}
}

func TestHasCycle(t *testing.T) {
	acyclic := graph.AdjacencyList{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}
	if dfs.HasCycle(acyclic) {
		t.Fatal("acyclic graph reported cycle")
	}

	cyclic := graph.AdjacencyList{
		"A": {"B"},
		"B": {"C"},
		"C": {"A"},
	}
	if !dfs.HasCycle(cyclic) {
		t.Fatal("cyclic graph missed cycle")
	}
}

func TestTraverseVisitFalseSkipsExpand(t *testing.T) {
	g := graph.AdjacencyList{
		"A": {"B"},
		"B": {"C"},
		"C": {},
	}
	var seen []string
	dfs.Traverse(g, "A", func(node string) bool {
		seen = append(seen, node)
		return node != "B"
	})
	if !reflect.DeepEqual(seen, []string{"A", "B"}) {
		t.Fatalf("seen = %v, want [A B]", seen)
	}
}
