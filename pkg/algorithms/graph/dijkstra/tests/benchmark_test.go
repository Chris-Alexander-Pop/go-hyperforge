package dijkstra_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph/dijkstra"
)

func BenchmarkShortestPath_LargeLinear(b *testing.B) {
	// Create a large linear graph: 0 -> 1 -> 2 -> ... -> N
	n := 2000
	g := make(dijkstra.Graph)
	for i := 0; i < n; i++ {
		u := fmt.Sprintf("%d", i)
		v := fmt.Sprintf("%d", i+1)
		g[u] = map[string]float64{v: 1.0}
	}
	// Add the last node to the graph so it exists as a key (optional but good for completeness)
	g[fmt.Sprintf("%d", n)] = map[string]float64{}

	start := "0"
	end := fmt.Sprintf("%d", n)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := dijkstra.ShortestPath(g, start, end)
		if result == nil {
			b.Fatalf("expected path, got nil")
		}
	}
}
