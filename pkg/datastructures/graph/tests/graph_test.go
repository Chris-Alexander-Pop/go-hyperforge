package graph_test

import (
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/graph"
)

func TestGraph_BFS(t *testing.T) {
	g := graph.New[string]()
	g.AddVertex("A")
	g.AddVertex("B")
	g.AddVertex("C")
	g.AddEdge("A", "B")
	g.AddEdge("A", "C")

	var order []string
	g.BFS("A", func(v string) { order = append(order, v) })
	if len(order) != 3 || order[0] != "A" {
		t.Fatalf("BFS order=%v want start A and 3 nodes", order)
	}
	seen := map[string]bool{}
	for _, v := range order {
		seen[v] = true
	}
	if !seen["B"] || !seen["C"] {
		t.Fatalf("BFS missed neighbors: %v", order)
	}
}

func TestGraph_DFS(t *testing.T) {
	g := graph.New[string]()
	g.AddVertex("A")
	g.AddVertex("B")
	g.AddVertex("C")
	g.AddEdge("A", "B")
	g.AddEdge("B", "C")

	var order []string
	g.DFS("A", func(v string) { order = append(order, v) })
	if len(order) != 3 || order[0] != "A" {
		t.Fatalf("DFS order=%v", order)
	}
}

func TestGraph_String(t *testing.T) {
	g := graph.New[string]()
	g.AddVertex("A")
	g.AddVertex("B")
	g.AddEdge("A", "B")
	s := g.String()
	if !strings.Contains(s, "A -> [B]") {
		t.Fatalf("String()=%q missing edge", s)
	}
}

func BenchmarkGraph_String(b *testing.B) {
	g := graph.New[int]()
	for i := 0; i < 100; i++ {
		g.AddVertex(i)
		for j := 0; j < 10; j++ {
			g.AddEdge(i, (i+j)%100)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.String()
	}
}
