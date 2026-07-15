package resilience

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Retry executes the function with automatic retries and exponential backoff.
func Retry(ctx context.Context, cfg RetryConfig, fn Executor) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 100 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.RetryIf == nil {
		cfg.RetryIf = func(err error) bool { return err != nil }
	}

	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Execute
		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !cfg.RetryIf(err) {
			return err
		}

		// Don't sleep after the last attempt
		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Calculate backoff with jitter
		jitter := 1.0
		if cfg.Jitter > 0 {
			jitter = 1.0 + (rand.Float64()*2-1)*cfg.Jitter
		}
		sleepDuration := time.Duration(float64(backoff) * jitter)

		// Sleep with context cancellation support
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
		}

		// Increase backoff for next iteration
		backoff = time.Duration(float64(backoff) * cfg.Multiplier)
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return lastErr
}

// RetryWithCircuitBreaker combines retry and circuit breaker.
func RetryWithCircuitBreaker(ctx context.Context, cb Breaker, retryCfg RetryConfig, fn Executor) error {
	return Retry(ctx, retryCfg, func(ctx context.Context) error {
		return cb.Execute(ctx, fn)
	})
}

// ExponentialBackoff calculates exponential backoff with jitter.
func ExponentialBackoff(attempt int, base time.Duration, max time.Duration, jitter float64) time.Duration {
	backoff := float64(base) * math.Pow(2, float64(attempt))

	if jitter > 0 {
		backoff *= 1.0 + (rand.Float64()*2-1)*jitter
	}

	if time.Duration(backoff) > max {
		return max
	}

	return time.Duration(backoff)
}

// WithTimeout wraps a function with a deadline.
//
// Behavior:
//   - timeout <= 0: runs fn with the parent context unchanged.
//   - On deadline: returns a CodeDeadlineExceeded AppError that unwraps to
//     context.DeadlineExceeded (errors.Is still works).
//   - Enforces the deadline even if fn ignores ctx by racing the call against
//     the timer. Prefer ctx-aware fns; a blocked fn may leave a goroutine until
//     it returns.
func WithTimeout(timeout time.Duration, fn Executor) Executor {
	return func(ctx context.Context) error {
		if timeout <= 0 {
			return fn(ctx)
		}

		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- fn(ctx)
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			select {
			case err := <-done:
				return err
			default:
				cause := ctx.Err()
				if errors.Is(cause, context.DeadlineExceeded) {
					return errors.DeadlineExceeded("operation timed out", cause)
				}
				return cause
			}
		}
	}
}
