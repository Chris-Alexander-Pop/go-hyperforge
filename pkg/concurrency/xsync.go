package concurrency

import (
	"context"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

// Re-export golang.org/x/sync primitives so callers can depend on pkg/concurrency
// without importing x/sync directly. The local Semaphore type remains the
// package's own weighted semaphore; NewWeighted wraps x/sync/semaphore.

// ErrGroup is an alias for errgroup.Group.
type ErrGroup = errgroup.Group

// ErrGroupWithContext returns a new Group and an associated Context derived from ctx
// (see errgroup.WithContext).
func ErrGroupWithContext(ctx context.Context) (*ErrGroup, context.Context) {
	return errgroup.WithContext(ctx)
}

// Weighted is an alias for semaphore.Weighted.
type Weighted = semaphore.Weighted

// NewWeighted creates a new weighted semaphore with the given maximum concurrent weight.
func NewWeighted(n int64) *Weighted {
	return semaphore.NewWeighted(n)
}
