package astar

import (
	"container/heap"
)

// Graph is a map of node -> neighbors (node -> weight).
type Graph map[string]map[string]float64

// Heuristic is a function that estimates distance between two nodes.
type Heuristic func(a, b string) float64

type PathResult struct {
	Distance float64
	Path     []string
}

// FindPath finds the shortest path using A*.
func FindPath(g Graph, start, end string, h Heuristic) *PathResult {
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{value: start, priority: 0}) // f = g + h, g=0 here

	gScore := make(map[string]float64)
	gScore[start] = 0

	fScore := make(map[string]float64)
	fScore[start] = h(start, end)

	previous := make(map[string]string)

	for pq.Len() > 0 {
		current := heap.Pop(pq).(*Item).value

		if current == end {
			// Reconstruct path
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
				fScore[neighbor] = f

				// In a real optimized A*, we update priority if exists.
				// Here we just push duplicate states, lazy deletion handled by checking processed?
				// Simple lazy approach: just push.
				heap.Push(pq, &Item{value: neighbor, priority: f})
			}
		}
	}
	return nil
}

// PQ shared logic (duplicated to stay self-contained module)
type Item struct {
	value    string
	priority float64
	index    int
}
type PriorityQueue []*Item

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq PriorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*pq = old[0 : n-1]
	return item
}
