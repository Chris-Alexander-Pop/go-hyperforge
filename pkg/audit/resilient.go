package audit

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientStore implements Store at compile time.
var _ Store = (*ResilientStore)(nil)

// ResilientConfig configures the resilient audit store wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientStore wraps a Store with circuit breaker and retry.
// Invalid-argument / not-supported outcomes are not retried and do not open the circuit.
type ResilientStore struct {
	next     Store
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientStore wraps next with resilience features.
func NewResilientStore(next Store, cfg ResilientConfig) *ResilientStore {
	rs := &ResilientStore{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rs.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "audit",
			FailureThreshold: threshold,
			SuccessThreshold: 2,
			Timeout:          timeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		rs.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				if errors.Is(err, ErrNotSupported) || errors.IsCode(err, CodeInvalidArgument) {
					return false
				}
				return true
			},
		}
	}

	return rs
}

func (s *ResilientStore) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if s.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := s.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if errors.Is(opErr, ErrNotSupported) || errors.IsCode(opErr, CodeInvalidArgument) {
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
	if s.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, s.retryCfg, operation)
	}
	return operation(ctx)
}

// Append runs Append with resilience.
func (s *ResilientStore) Append(ctx context.Context, event Event) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.Append(ctx, event)
	})
}

// Query runs Query with resilience.
func (s *ResilientStore) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	var events []Event
	err := s.execute(ctx, func(ctx context.Context) error {
		var e error
		events, e = s.next.Query(ctx, filter)
		return e
	})
	return events, err
}

// Unwrap returns the underlying store.
func (s *ResilientStore) Unwrap() Store {
	return s.next
}
