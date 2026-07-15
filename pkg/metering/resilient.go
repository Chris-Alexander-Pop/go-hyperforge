package metering

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientMeter implements Meter at compile time.
var _ Meter = (*ResilientMeter)(nil)

// ResilientConfig configures the resilient metering wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientMeter wraps a Meter with circuit breaker and retry.
// Invalid usage / closed outcomes are not retried and do not open the circuit.
type ResilientMeter struct {
	next     Meter
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientMeter wraps next with resilience features.
func NewResilientMeter(next Meter, cfg ResilientConfig) *ResilientMeter {
	rm := &ResilientMeter{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rm.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "metering",
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
		rm.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				if errors.Is(err, ErrInvalidUsage) || errors.IsCode(err, CodeClosed) {
					return false
				}
				return true
			},
		}
	}

	return rm
}

func (m *ResilientMeter) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if m.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := m.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if errors.Is(opErr, ErrInvalidUsage) || errors.IsCode(opErr, CodeClosed) {
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
	if m.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, m.retryCfg, operation)
	}
	return operation(ctx)
}

// RecordUsage runs RecordUsage with resilience.
func (m *ResilientMeter) RecordUsage(ctx context.Context, event UsageEvent) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.RecordUsage(ctx, event)
	})
}

// GetUsage runs GetUsage with resilience.
func (m *ResilientMeter) GetUsage(ctx context.Context, filter UsageFilter) ([]UsageEvent, error) {
	var events []UsageEvent
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		events, e = m.next.GetUsage(ctx, filter)
		return e
	})
	return events, err
}

// PeriodAggregate runs PeriodAggregate with resilience.
func (m *ResilientMeter) PeriodAggregate(ctx context.Context, filter UsageFilter, period time.Duration) ([]PeriodBucket, error) {
	var buckets []PeriodBucket
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		buckets, e = m.next.PeriodAggregate(ctx, filter, period)
		return e
	})
	return buckets, err
}

// SummarizeUsage runs SummarizeUsage with resilience.
func (m *ResilientMeter) SummarizeUsage(ctx context.Context, filter UsageFilter) (*UsageSummary, error) {
	var summary *UsageSummary
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		summary, e = m.next.SummarizeUsage(ctx, filter)
		return e
	})
	return summary, err
}

// Close releases resources held by the underlying meter.
func (m *ResilientMeter) Close() error {
	return m.next.Close()
}

// Unwrap returns the underlying meter.
func (m *ResilientMeter) Unwrap() Meter {
	return m.next
}
