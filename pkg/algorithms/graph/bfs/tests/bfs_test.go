package bfs_test

import (
	"reflect"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph"
	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph/bfs"
)

func sampleGraph() graph.AdjacencyList {
	return graph.AdjacencyList{
		"A": {"B", "C"},
		"B": {"D"},
		"C": {"D", "E"},
		"D": {},
		"E": {},
	}
}

func TestReachable(t *testing.T) {
	got := bfs.Reachable(sampleGraph(), "A")
	want := []string{"A", "B", "C", "D", "E"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reachable = %v, want %v", got, want)
	}
}

func TestShortestPath(t *testing.T) {
	g := sampleGraph()
	path := bfs.ShortestPath(g, "A", "D")
	want := []string{"A", "B", "D"}
	if !reflect.DeepEqual(path, want) {
		t.Fatalf("ShortestPath A→D = %v, want %v", path, want)
	}

	if bfs.ShortestPath(g, "A", "Z") != nil {
		t.Fatal("unreachable should return nil")
	}

	if got := bfs.ShortestPath(g, "A", "A"); !reflect.DeepEqual(got, []string{"A"}) {
		t.Fatalf("ShortestPath A→A = %v, want [A]", got)
	}
}

func TestTraverseEarlyStop(t *testing.T) {
	var seen []string
	bfs.Traverse(sampleGraph(), "A", func(node string) bool {
		seen = append(seen, node)
		return node != "B"
	})
	if !reflect.DeepEqual(seen, []string{"A", "B"}) {
		t.Fatalf("early stop seen = %v, want [A B]", seen)
	}
}
