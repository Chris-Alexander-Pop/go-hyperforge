package dijkstra

import (
	"container/heap"
	"math"
)

// Graph is a map of node -> neighbors (node -> weight).
type Graph map[string]map[string]float64

// PathResult contains the distance and path to a target.
type PathResult struct {
	Distance float64
	Path     []string
}

// ShortestPath finds the shortest path from start to end.
func ShortestPath(g Graph, start, end string) *PathResult {
	// Priority Queue
	pq := &PriorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &Item{value: start, priority: 0})

	distances := make(map[string]float64)
	distances[start] = 0
	previous := make(map[string]string)

	for pq.Len() > 0 {
		u := heap.Pop(pq).(*Item).value

		if u == end {
			// Construct path
			path := []string{}
			curr := end
			for curr != "" {
				path = append([]string{curr}, path...) // Prepend
				curr = previous[curr]
				if curr == start {
					path = append([]string{start}, path...)
					break
				}
			}
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
				heap.Push(pq, &Item{value: v, priority: alt})
			}
		}
	}

	return nil // Not reachable
}

// PQ implementation
type Item struct {
	value    string
	priority float64
	index    int
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int           { return len(pq) }
func (pq PriorityQueue) Less(i, j int) bool { return pq[i].priority < pq[j].priority }
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
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
