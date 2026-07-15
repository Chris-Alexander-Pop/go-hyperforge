package graph

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrVertexNotFound indicates that the vertex was not found.
	ErrVertexNotFound = errors.NotFound

	// ErrEdgeNotFound indicates that the edge was not found.
	ErrEdgeNotFound = errors.NotFound

	// ErrInvalidQuery indicates an invalid query syntax.
	ErrInvalidQuery = errors.InvalidArgument
)
