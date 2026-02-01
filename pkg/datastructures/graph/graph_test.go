package graph

import (
	"strings"
	"testing"
)

func BenchmarkGraph_String(b *testing.B) {
	g := New[int]()
	// Create a graph with some complexity
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

func TestGraph_String(t *testing.T) {
	g := New[string]()
	g.AddVertex("A")
	g.AddVertex("B")
	g.AddEdge("A", "B")

	s := g.String()
	// Map iteration order is random, so exact string match is hard.
	// We check if it contains expected parts.
	expectedPart := "A -> [B]"
	if !strings.Contains(s, expectedPart) {
		t.Errorf("Expected string to contain %q, but got %q", expectedPart, s)
	}

}
