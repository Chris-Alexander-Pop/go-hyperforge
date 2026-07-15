package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Store implements graph.Interface with an in-memory store.
type Store struct {
	vertices map[string]*graph.Vertex
	edges    map[string]*graph.Edge
	mu       *concurrency.SmartRWMutex
}

// New creates a new in-memory graph store.
func New() *Store {
	return &Store{
		vertices: make(map[string]*graph.Vertex),
		edges:    make(map[string]*graph.Edge),
		mu:       concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-graph"}),
	}
}

// AddVertex adds a vertex to the graph.
func (s *Store) AddVertex(ctx context.Context, v *graph.Vertex) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.vertices[v.ID] = cloneVertex(v)
	return nil
}

// AddEdge adds an edge to the graph.
func (s *Store) AddEdge(ctx context.Context, e *graph.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify vertices exist
	if _, ok := s.vertices[e.FromID]; !ok {
		return errors.NotFound("source vertex not found", nil)
	}
	if _, ok := s.vertices[e.ToID]; !ok {
		return errors.NotFound("target vertex not found", nil)
	}

	s.edges[e.ID] = cloneEdge(e)
	return nil
}

// GetVertex retrieves a vertex by ID.
func (s *Store) GetVertex(ctx context.Context, id string) (*graph.Vertex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.vertices[id]
	if !ok {
		return nil, errors.NotFound("vertex not found", nil)
	}
	return cloneVertex(v), nil
}

// GetNeighbors retrieves neighbor vertices connected by an edge with the given label.
func (s *Store) GetNeighbors(ctx context.Context, vertexID string, edgeLabel string, direction string) ([]*graph.Vertex, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var neighbors []*graph.Vertex

	for _, edge := range s.edges {
		if edgeLabel != "" && edge.Label != edgeLabel {
			continue
		}

		var targetID string
		switch direction {
		case "out", "outgoing":
			if edge.FromID == vertexID {
				targetID = edge.ToID
			}
		case "in", "incoming":
			if edge.ToID == vertexID {
				targetID = edge.FromID
			}
		default: // both
			if edge.FromID == vertexID {
				targetID = edge.ToID
			} else if edge.ToID == vertexID {
				targetID = edge.FromID
			}
		}

		if targetID != "" {
			if v, ok := s.vertices[targetID]; ok {
				neighbors = append(neighbors, cloneVertex(v))
			}
		}
	}

	return neighbors, nil
}

// Query executes a query (not implemented for in-memory).
func (s *Store) Query(ctx context.Context, query string, args map[string]interface{}) (interface{}, error) {
	return nil, errors.New(errors.CodeInternal, "query not supported in memory adapter", nil)
}

// Close clears the in-memory store.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vertices = make(map[string]*graph.Vertex)
	s.edges = make(map[string]*graph.Edge)
	return nil
}

func cloneVertex(v *graph.Vertex) *graph.Vertex {
	props := make(map[string]interface{})
	for k, val := range v.Properties {
		props[k] = val
	}
	return &graph.Vertex{
		ID:         v.ID,
		Label:      v.Label,
		Properties: props,
	}
}

func cloneEdge(e *graph.Edge) *graph.Edge {
	props := make(map[string]interface{})
	for k, val := range e.Properties {
		props[k] = val
	}
	return &graph.Edge{
		ID:         e.ID,
		Label:      e.Label,
		FromID:     e.FromID,
		ToID:       e.ToID,
		Properties: props,
	}
}

// Ensure Store implements graph.Interface
var _ graph.Interface = (*Store)(nil)
