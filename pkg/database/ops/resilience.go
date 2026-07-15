package ops

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// WithRetry executes the operation with exponential backoff retries.
// It delegates to pkg/resilience.Retry while preserving the historical
// (ctx, attempts, backoff, op) signature.
//
// Useful for transient network errors or db connection glitches.
func WithRetry(ctx context.Context, attempts int, backoff time.Duration, op func() error) error {
	if attempts <= 0 {
		attempts = 1
	}
	if backoff <= 0 {
		backoff = 100 * time.Millisecond
	}

	cfg := resilience.RetryConfig{
		MaxAttempts:    attempts,
		InitialBackoff: backoff,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		Jitter:         0,
		RetryIf:        func(err error) bool { return err != nil },
	}

	err := resilience.Retry(ctx, cfg, func(ctx context.Context) error {
		return op()
	})
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return errors.Wrap(err, "max retries exceeded")
}
