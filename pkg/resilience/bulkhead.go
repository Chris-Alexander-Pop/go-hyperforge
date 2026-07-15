package resilience

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Bulkhead isolates concurrent work behind a semaphore so failures in one
// partition cannot exhaust shared resources.
type Bulkhead struct {
	name string
	sem  *concurrency.Semaphore
	max  int64
}

// BulkheadConfig configures a bulkhead.
type BulkheadConfig struct {
	// Name identifies this bulkhead (for logging/metrics).
	Name string

	// MaxConcurrent is the maximum number of in-flight executions.
	MaxConcurrent int64
}

// NewBulkhead creates a semaphore-bounded bulkhead.
func NewBulkhead(cfg BulkheadConfig) *Bulkhead {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	return &Bulkhead{
		name: cfg.Name,
		sem:  concurrency.NewSemaphore(cfg.MaxConcurrent),
		max:  cfg.MaxConcurrent,
	}
}

// Name returns the bulkhead name.
func (b *Bulkhead) Name() string {
	return b.name
}

// MaxConcurrent returns the configured concurrency limit.
func (b *Bulkhead) MaxConcurrent() int64 {
	return b.max
}

// Execute acquires a slot, runs fn, then releases the slot.
// If ctx is cancelled while waiting, it returns ctx.Err().
func (b *Bulkhead) Execute(ctx context.Context, fn Executor) error {
	if err := b.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer b.sem.Release(1)
	return fn(ctx)
}

// TryExecute attempts a non-blocking acquire. Returns ErrBulkheadFull if no
// slot is available.
func (b *Bulkhead) TryExecute(ctx context.Context, fn Executor) error {
	if !b.sem.TryAcquire(1) {
		return ErrBulkheadFull
	}
	defer b.sem.Release(1)
	return fn(ctx)
}
