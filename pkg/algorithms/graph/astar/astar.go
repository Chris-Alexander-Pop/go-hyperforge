package astar

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/heap"
)

// Graph is a map of node -> neighbors (node -> weight).
type Graph map[string]map[string]float64

// Heuristic is a function that estimates distance between two nodes.
type Heuristic func(a, b string) float64

// PathResult contains the distance and reconstructed path.
type PathResult struct {
	Distance float64
	Path     []string
}

// FindPath finds the shortest path using A* with pkg/datastructures/heap.
func FindPath(g Graph, start, end string, h Heuristic) *PathResult {
	pq := heap.NewMinHeap[string]()
	pq.PushItem(start, h(start, end))

	gScore := make(map[string]float64)
	gScore[start] = 0

	previous := make(map[string]string)

	for pq.Size() > 0 {
		current, _, ok := pq.PopItem()
		if !ok {
			break
		}

		if current == end {
			path := []string{end}
			curr := end
			for {
				prev, ok := previous[curr]
				if !ok {
					break
				}
				path = append([]string{prev}, path...)
				curr = prev
			}
			return &PathResult{Distance: gScore[end], Path: path}
		}

		for neighbor, weight := range g[current] {
			tentativeG := gScore[current] + weight

			if val, ok := gScore[neighbor]; !ok || tentativeG < val {
				previous[neighbor] = current
				gScore[neighbor] = tentativeG
				f := tentativeG + h(neighbor, end)
				// Lazy duplicate entries; first time a node is expanded with
				// its best score wins (standard A* heap relaxation).
				pq.PushItem(neighbor, f)
			}
		}
	}
	return nil
}
