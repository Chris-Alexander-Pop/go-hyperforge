package secrets

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientSecretManager implements SecretManager at compile time.
var _ SecretManager = (*ResilientSecretManager)(nil)

// ResilientConfig configures the resilient secrets wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientSecretManager wraps a SecretManager with circuit breaker and retry.
// NotFound / invalid-argument outcomes are not retried and do not open the circuit.
type ResilientSecretManager struct {
	next     SecretManager
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientSecretManager wraps next with resilience features.
func NewResilientSecretManager(next SecretManager, cfg ResilientConfig) *ResilientSecretManager {
	rm := &ResilientSecretManager{next: next}

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
			Name:             "secrets",
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
				case ErrNotFound, ErrInvalidArgument:
					return false
				}
				return true
			},
		}
	}

	return rm
}

func (m *ResilientSecretManager) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if m.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := m.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if opErr == ErrNotFound || opErr == ErrInvalidArgument {
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

// Get runs Get with resilience.
func (m *ResilientSecretManager) Get(ctx context.Context, name string) (string, error) {
	var val string
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		val, e = m.next.Get(ctx, name)
		return e
	})
	return val, err
}

// Set runs Set with resilience.
func (m *ResilientSecretManager) Set(ctx context.Context, name, value string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.Set(ctx, name, value)
	})
}

// Rotate runs Rotate with resilience.
func (m *ResilientSecretManager) Rotate(ctx context.Context, name, newValue string) (string, error) {
	var val string
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		val, e = m.next.Rotate(ctx, name, newValue)
		return e
	})
	return val, err
}

// Unwrap returns the underlying manager.
func (m *ResilientSecretManager) Unwrap() SecretManager {
	return m.next
}
