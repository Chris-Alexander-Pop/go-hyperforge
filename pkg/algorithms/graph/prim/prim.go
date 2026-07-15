package prim

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/heap"
)

// Graph: node -> neighbor -> weight
type Graph map[string]map[string]float64

// Edge is an MST edge.
type Edge struct {
	U, V   string
	Weight float64
}

type pqEdge struct {
	U, V string
}

// MST finds Minimum Spanning Tree using Prim's Algorithm and
// pkg/datastructures/heap.
func MST(g Graph) ([]Edge, float64) {
	if len(g) == 0 {
		return nil, 0
	}

	var start string
	for k := range g {
		start = k
		break
	}

	visited := make(map[string]bool)
	visited[start] = true

	pq := heap.NewMinHeap[pqEdge]()
	for neighbor, weight := range g[start] {
		pq.PushItem(pqEdge{U: start, V: neighbor}, weight)
	}

	var result []Edge
	var totalWeight float64

	for pq.Size() > 0 {
		e, w, ok := pq.PopItem()
		if !ok {
			break
		}
		if visited[e.V] {
			continue
		}

		visited[e.V] = true
		result = append(result, Edge{U: e.U, V: e.V, Weight: w})
		totalWeight += w

		for next, weight := range g[e.V] {
			if !visited[next] {
				pq.PushItem(pqEdge{U: e.V, V: next}, weight)
			}
		}
	}

	return result, totalWeight
}
