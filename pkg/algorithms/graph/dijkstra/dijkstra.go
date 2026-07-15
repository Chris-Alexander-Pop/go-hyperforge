package dijkstra

import (
	"math"
	"slices"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
)

// Graph is a map of node -> neighbors (node -> weight).
type Graph map[string]map[string]float64

// PathResult contains the distance and path to a target.
type PathResult struct {
	Distance float64
	Path     []string
}

// ShortestPath finds the shortest path from start to end using a min-heap
// from pkg/datastructures/heap.
func ShortestPath(g Graph, start, end string) *PathResult {
	pq := heap.NewMinHeap[string]()
	pq.PushItem(start, 0)

	distances := make(map[string]float64)
	distances[start] = 0
	previous := make(map[string]string)

	for pq.Size() > 0 {
		u, _, ok := pq.PopItem()
		if !ok {
			break
		}

		if u == end {
			path := []string{}
			curr := end
			for curr != "" {
				path = append(path, curr)
				curr = previous[curr]
				if curr == start {
					path = append(path, start)
					break
				}
			}
			slices.Reverse(path)
			return &PathResult{Distance: distances[end], Path: path}
		}

		if d, ok := distances[u]; ok && d == math.MaxFloat64 {
			continue
		}

		for v, weight := range g[u] {
			alt := distances[u] + weight
			if dist, ok := distances[v]; !ok || alt < dist {
				distances[v] = alt
				previous[v] = u
				pq.PushItem(v, alt)
			}
		}
	}

	return nil
}
