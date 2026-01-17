package prim

import (
	"container/heap"
)

// Graph: node -> neighbor -> weight
type Graph map[string]map[string]float64

type Edge struct {
	U, V   string
	Weight float64
}

// MST finds Minimum Spanning Tree using Prim's Algorithm.
func MST(g Graph) ([]Edge, float64) {
	if len(g) == 0 {
		return nil, 0
	}

	// Pick arbitrary start node
	var start string
	for k := range g {
		start = k
		break
	}

	visited := make(map[string]bool)
	visited[start] = true

	pq := &PriorityQueue{}
	heap.Init(pq)

	// Add initial edges
	for neighbor, weight := range g[start] {
		heap.Push(pq, &Item{u: start, v: neighbor, priority: weight})
	}

	var result []Edge
	var totalWeight float64
	// Total nodes we need to connect: len(g)
	// MST has V-1 edges.

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*Item)
		u, v, w := item.u, item.v, item.priority

		if visited[v] {
			continue
		}

		visited[v] = true
		result = append(result, Edge{U: u, V: v, Weight: w})
		totalWeight += w

		for next, weight := range g[v] {
			if !visited[next] {
				heap.Push(pq, &Item{u: v, v: next, priority: weight})
			}
		}
	}

	return result, totalWeight
}

// PQ
type Item struct {
	u, v     string
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
