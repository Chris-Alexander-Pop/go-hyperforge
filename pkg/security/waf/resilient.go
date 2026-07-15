package waf

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientManager implements Manager at compile time.
var _ Manager = (*ResilientManager)(nil)

// ResilientConfig configures the resilient WAF wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientManager wraps a Manager with circuit breaker and retry.
// Invalid-rule / not-found outcomes are not retried and do not open the circuit.
type ResilientManager struct {
	next     Manager
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientManager wraps next with resilience features.
func NewResilientManager(next Manager, cfg ResilientConfig) *ResilientManager {
	rm := &ResilientManager{next: next}

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
			Name:             "waf",
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
				switch err {
				case ErrInvalidRule, ErrNotFound:
					return false
				}
				return true
			},
		}
	}

	return rm
}

func (m *ResilientManager) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if m.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := m.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if opErr == ErrInvalidRule || opErr == ErrNotFound {
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

// BlockIP runs BlockIP with resilience.
func (m *ResilientManager) BlockIP(ctx context.Context, ip, reason string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.BlockIP(ctx, ip, reason)
	})
}

// AllowIP runs AllowIP with resilience.
func (m *ResilientManager) AllowIP(ctx context.Context, ip string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.AllowIP(ctx, ip)
	})
}

// GetRules runs GetRules with resilience.
func (m *ResilientManager) GetRules(ctx context.Context) ([]Rule, error) {
	var rules []Rule
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		rules, e = m.next.GetRules(ctx)
		return e
	})
	return rules, err
}

// Unwrap returns the underlying manager.
func (m *ResilientManager) Unwrap() Manager {
	return m.next
}
