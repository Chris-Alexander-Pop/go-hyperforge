package testsuite

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// GraphSuite is a reusable conformance suite for graph.Interface implementations.
type GraphSuite struct {
	*test.Suite
	Store graph.Interface
	// Optional cleanup after each test.
	Cleanup func()
}

func (s *GraphSuite) TearDownTest() {
	if s.Cleanup != nil {
		s.Cleanup()
	}
}

func (s *GraphSuite) TestAddGetVertex() {
	ctx := context.Background()

	v := &graph.Vertex{
		ID:         "v1",
		Label:      "Person",
		Properties: map[string]interface{}{"name": "Alice"},
	}
	s.NoError(s.Store.AddVertex(ctx, v))

	got, err := s.Store.GetVertex(ctx, "v1")
	s.NoError(err)
	s.Equal("v1", got.ID)
	s.Equal("Person", got.Label)
	s.Equal("Alice", got.Properties["name"])
}

func (s *GraphSuite) TestGetMissingVertex() {
	ctx := context.Background()
	_, err := s.Store.GetVertex(ctx, "missing")
	s.Error(err)
	var appErr *errors.AppError
	if errors.As(err, &appErr) {
		s.Equal(errors.CodeNotFound, appErr.Code)
	}
}

func (s *GraphSuite) TestAddEdgeAndNeighbors() {
	ctx := context.Background()

	s.NoError(s.Store.AddVertex(ctx, &graph.Vertex{ID: "a", Label: "Person"}))
	s.NoError(s.Store.AddVertex(ctx, &graph.Vertex{ID: "b", Label: "Person"}))
	s.NoError(s.Store.AddEdge(ctx, &graph.Edge{
		ID:     "e1",
		Label:  "KNOWS",
		FromID: "a",
		ToID:   "b",
	}))

	neighbors, err := s.Store.GetNeighbors(ctx, "a", "KNOWS", "out")
	s.NoError(err)
	s.Len(neighbors, 1)
	s.Equal("b", neighbors[0].ID)

	neighbors, err = s.Store.GetNeighbors(ctx, "b", "KNOWS", "in")
	s.NoError(err)
	s.Len(neighbors, 1)
	s.Equal("a", neighbors[0].ID)
}

func (s *GraphSuite) TestAddEdgeMissingVertex() {
	ctx := context.Background()

	s.NoError(s.Store.AddVertex(ctx, &graph.Vertex{ID: "only", Label: "Node"}))

	err := s.Store.AddEdge(ctx, &graph.Edge{
		ID:     "bad",
		Label:  "LINK",
		FromID: "only",
		ToID:   "missing",
	})
	s.Error(err)
}

func (s *GraphSuite) TestClose() {
	s.NoError(s.Store.Close())
	s.Cleanup = nil
}
