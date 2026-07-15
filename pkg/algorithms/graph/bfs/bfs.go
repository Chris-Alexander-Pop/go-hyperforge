package bfs

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/graph"
)

// Visit reports whether traversal should continue. Return false to stop early.
type Visit func(node string) bool

// Traverse runs BFS from start, calling visit for each newly discovered node
// (including start). Order is level-order. Neighbors of a node are visited in
// the order they appear in the adjacency list.
func Traverse(g graph.AdjacencyList, start string, visit Visit) {
	if visit == nil {
		return
	}
	visited := make(map[string]bool, len(g))
	queue := []string{start}
	visited[start] = true

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		if !visit(u) {
			return
		}
		for _, v := range g[u] {
			if !visited[v] {
				visited[v] = true
				queue = append(queue, v)
			}
		}
	}
}

// Reachable returns all nodes reachable from start, including start, in BFS order.
func Reachable(g graph.AdjacencyList, start string) []string {
	var out []string
	Traverse(g, start, func(node string) bool {
		out = append(out, node)
		return true
	})
	return out
}

// ShortestPath returns an unweighted shortest path from start to end.
// Returns nil if end is unreachable (or start/end empty with no path).
func ShortestPath(g graph.AdjacencyList, start, end string) []string {
	if start == end {
		return []string{start}
	}

	visited := make(map[string]bool, len(g))
	parent := make(map[string]string, len(g))
	queue := []string{start}
	visited[start] = true

	found := false
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		if u == end {
			found = true
			break
		}
		for _, v := range g[u] {
			if !visited[v] {
				visited[v] = true
				parent[v] = u
				queue = append(queue, v)
			}
		}
	}
	if !found {
		return nil
	}

	path := []string{end}
	for cur := end; cur != start; {
		p, ok := parent[cur]
		if !ok {
			return nil
		}
		path = append(path, p)
		cur = p
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}
