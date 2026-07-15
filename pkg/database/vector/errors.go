package vector

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for vector database operations.
var (
	// ErrVectorNotFound is returned when a vector ID does not exist.
	ErrVectorNotFound = errors.New(errors.CodeNotFound, "vector not found", nil)

	// ErrSearchFailed is returned when a search operation fails.
	ErrSearchFailed = errors.New(errors.CodeInternal, "vector search failed", nil)

	// ErrUpsertFailed is returned when an upsert operation fails.
	ErrUpsertFailed = errors.New(errors.CodeInternal, "vector upsert failed", nil)

	// ErrInvalidDimension is returned when vector dimensions don't match.
	ErrInvalidDimension = errors.New(errors.CodeInvalidArgument, "invalid vector dimension", nil)
)
