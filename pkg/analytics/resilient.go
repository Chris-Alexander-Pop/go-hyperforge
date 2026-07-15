package analytics

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientTracker implements Tracker at compile time.
var _ Tracker = (*ResilientTracker)(nil)

// ResilientConfig configures the resilient analytics tracker wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientTracker wraps a Tracker with circuit breaker and retry.
// NotFound / closed outcomes are not retried and do not open the circuit.
type ResilientTracker struct {
	next     Tracker
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientTracker wraps next with resilience features.
func NewResilientTracker(next Tracker, cfg ResilientConfig) *ResilientTracker {
	rt := &ResilientTracker{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rt.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "analytics",
			FailureThreshold: threshold,
			SuccessThreshold: 2,
			Timeout:          timeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 50 * time.Millisecond
		}
		rt.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				if IsNotFound(err) || errors.Is(err, ErrClosed) {
					return false
				}
				return true
			},
		}
	}

	return rt
}

func (t *ResilientTracker) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if t.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := t.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if IsNotFound(opErr) || errors.Is(opErr, ErrClosed) {
					return nil
				}
				return opErr
			})
			if cbErr != nil {
				return cbErr
			}
			return opErr
		}
	}
	if t.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, t.retryCfg, operation)
	}
	return operation(ctx)
}

// Add runs Add with resilience.
func (t *ResilientTracker) Add(ctx context.Context, counter string, element string) error {
	return t.execute(ctx, func(ctx context.Context) error {
		return t.next.Add(ctx, counter, element)
	})
}

// Count runs Count with resilience.
func (t *ResilientTracker) Count(ctx context.Context, counter string) (uint64, error) {
	var n uint64
	err := t.execute(ctx, func(ctx context.Context) error {
		var e error
		n, e = t.next.Count(ctx, counter)
		return e
	})
	return n, err
}

// Reset runs Reset with resilience.
func (t *ResilientTracker) Reset(ctx context.Context, counter string) error {
	return t.execute(ctx, func(ctx context.Context) error {
		return t.next.Reset(ctx, counter)
	})
}

// Merge runs Merge with resilience.
func (t *ResilientTracker) Merge(ctx context.Context, dest, source string) error {
	return t.execute(ctx, func(ctx context.Context) error {
		return t.next.Merge(ctx, dest, source)
	})
}

// Close releases resources held by the underlying tracker.
func (t *ResilientTracker) Close() error {
	return t.next.Close()
}

// Unwrap returns the underlying tracker.
func (t *ResilientTracker) Unwrap() Tracker {
	return t.next
}
