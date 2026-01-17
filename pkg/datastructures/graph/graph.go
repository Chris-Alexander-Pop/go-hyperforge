package graph

import (
	"fmt"
	"sync"
)

// Graph represents a generic graph using adjacency list.
type Graph[T comparable] struct {
	adj map[T][]T
	mu  sync.RWMutex
}

func New[T comparable]() *Graph[T] {
	return &Graph[T]{
		adj: make(map[T][]T),
	}
}

func (g *Graph[T]) AddVertex(v T) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.adj[v]; !exists {
		g.adj[v] = []T{}
	}
}

func (g *Graph[T]) AddEdge(u, v T) {
	g.mu.Lock()
	defer g.mu.Unlock()
	// Undirected or Directed? Let's assume Directed by default.
	// For Undirected, call twice manually or add helper.
	g.adj[u] = append(g.adj[u], v)
}

func (g *Graph[T]) BFS(start T, visit func(T)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[T]bool)
	queue := []T{start}
	visited[start] = true

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		visit(u)

		for _, v := range g.adj[u] {
			if !visited[v] {
				visited[v] = true
				queue = append(queue, v)
			}
		}
	}
}

func (g *Graph[T]) DFS(start T, visit func(T)) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[T]bool)
	var dfs func(T)
	dfs = func(u T) {
		visited[u] = true
		visit(u)
		for _, v := range g.adj[u] {
			if !visited[v] {
				dfs(v)
			}
		}
	}
	if _, ok := g.adj[start]; ok {
		dfs(start)
	}
}

func (g *Graph[T]) String() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	s := ""
	for u, neighbors := range g.adj {
		s += fmt.Sprintf("%v -> %v\n", u, neighbors)
	}
	return s
}
