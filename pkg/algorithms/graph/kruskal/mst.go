package kruskal

import (
	"sort"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/set/disjoint"
)

// Edge represents a graph edge.
type Edge struct {
	U, V   string
	Weight float64
}

// MST finds the Minimum Spanning Tree using Kruskal's algorithm.
func MST(edges []Edge) ([]Edge, float64) {
	// Sort edges by weight
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Weight < edges[j].Weight
	})

	ds := disjoint.New()

	// Pre-populate disjoint set with all vertices
	for _, e := range edges {
		ds.MakeSet(e.U)
		ds.MakeSet(e.V)
	}

	var result []Edge
	var totalWeight float64

	for _, e := range edges {
		if !ds.Connected(e.U, e.V) {
			ds.Union(e.U, e.V)
			result = append(result, e)
			totalWeight += e.Weight
		}
	}

	return result, totalWeight
}
