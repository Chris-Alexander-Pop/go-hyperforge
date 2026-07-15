package dfs

import (
	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/graph"
)

// Visit reports whether traversal should continue into the node's neighbors.
// Return false to skip expanding this node (siblings may still be visited).
type Visit func(node string) bool

// Traverse runs iterative DFS from start, calling visit for each newly
// discovered node (including start). Neighbor order follows the adjacency list;
// later neighbors are explored first (stack LIFO).
func Traverse(g graph.AdjacencyList, start string, visit Visit) {
	if visit == nil {
		return
	}
	visited := make(map[string]bool, len(g))
	stack := []string{start}

	for len(stack) > 0 {
		u := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[u] {
			continue
		}
		visited[u] = true
		if !visit(u) {
			continue
		}
		neighbors := g[u]
		for i := len(neighbors) - 1; i >= 0; i-- {
			v := neighbors[i]
			if !visited[v] {
				stack = append(stack, v)
			}
		}
	}
}

// Reachable returns all nodes reachable from start, including start, in DFS order.
func Reachable(g graph.AdjacencyList, start string) []string {
	var out []string
	Traverse(g, start, func(node string) bool {
		out = append(out, node)
		return true
	})
	return out
}

// HasCycle reports whether the directed graph contains a cycle reachable from
// any node present as a key in g.
func HasCycle(g graph.AdjacencyList) bool {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[string]int, len(g))

	var visit func(string) bool
	visit = func(u string) bool {
		color[u] = gray
		for _, v := range g[u] {
			switch color[v] {
			case gray:
				return true
			case white:
				if visit(v) {
					return true
				}
			}
		}
		color[u] = black
		return false
	}

	for u := range g {
		if color[u] == white && visit(u) {
			return true
		}
	}
	return false
}
