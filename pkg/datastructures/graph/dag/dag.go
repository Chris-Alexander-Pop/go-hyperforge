package dag

import (
	"errors"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

var (
	ErrCycleDetected = errors.New("cycle detected")
	ErrNodeNotFound  = errors.New("node not found")
)

// DAG represents a Directed Acyclic Graph.
type DAG[T any] struct {
	nodes map[string]*node[T]
	mu    *concurrency.SmartRWMutex
}

type node[T any] struct {
	id    string
	value T
	in    map[string]struct{} // incoming edges (dependencies)
	out   map[string]struct{} // outgoing edges (dependents)
}

func New[T any]() *DAG[T] {
	return &DAG[T]{
		nodes: make(map[string]*node[T]),
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "DAG"}),
	}
}

// AddNode adds a node.
func (g *DAG[T]) AddNode(id string, value T) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.nodes[id]; !exists {
		g.nodes[id] = &node[T]{
			id:    id,
			value: value,
			in:    make(map[string]struct{}),
			out:   make(map[string]struct{}),
		}
	}
}

// AddEdge adds a directed edge from 'from' to 'to' (from -> to).
// Meaning 'to' depends on 'from'.
func (g *DAG[T]) AddEdge(from, to string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	fNode, ok1 := g.nodes[from]
	tNode, ok2 := g.nodes[to]

	if !ok1 || !ok2 {
		return ErrNodeNotFound
	}

	// Check for self-loop
	if from == to {
		return ErrCycleDetected
	}

	// Tentatively add
	fNode.out[to] = struct{}{}
	tNode.in[from] = struct{}{}

	// Check Cycle
	if g.detectCycle() {
		// Rollback
		delete(fNode.out, to)
		delete(tNode.in, from)
		return ErrCycleDetected
	}
	return nil
}

// TopologicalSort returns a valid ordering.
func (g *DAG[T]) TopologicalSort() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Kahns Algorithm
	inDegree := make(map[string]int)
	for id, n := range g.nodes {
		inDegree[id] = len(n.in)
	}

	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	count := 0

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]
		result = append(result, currID)
		count++

		currNode := g.nodes[currID]
		for neighbor := range currNode.out {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if count != len(g.nodes) {
		return nil, ErrCycleDetected
	}
	return result, nil
}

// detectCycle checks if graph has cycle (DFS).
// Note: This is O(V+E), called on every edge add is expensive O((V+E)*E).
// For production, simpler checks or delayed validation is better.
func (g *DAG[T]) detectCycle() bool {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var hasCycle bool
	var visit func(string)

	visit = func(id string) {
		if hasCycle {
			return
		}
		visited[id] = true
		recursionStack[id] = true

		if n, ok := g.nodes[id]; ok {
			for neighbor := range n.out {
				if !visited[neighbor] {
					visit(neighbor)
				} else if recursionStack[neighbor] {
					hasCycle = true
					return
				}
			}
		}
		recursionStack[id] = false
	}

	for id := range g.nodes {
		if !visited[id] {
			visit(id)
			if hasCycle {
				return true
			}
		}
	}
	return false
}
