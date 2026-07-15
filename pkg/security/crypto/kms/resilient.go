package kms

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientKeyManager implements KeyManager at compile time.
var _ KeyManager = (*ResilientKeyManager)(nil)

// ResilientConfig configures the resilient KMS wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientKeyManager wraps a KeyManager with circuit breaker and retry.
// Invalid-argument / not-supported outcomes are not retried and do not open the circuit.
type ResilientKeyManager struct {
	next     KeyManager
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientKeyManager wraps next with resilience features.
func NewResilientKeyManager(next KeyManager, cfg ResilientConfig) *ResilientKeyManager {
	rm := &ResilientKeyManager{next: next}

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
			Name:             "kms",
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
				case ErrInvalidArgument, ErrNotSupported:
					return false
				}
				return true
			},
		}
	}

	return rm
}

func (m *ResilientKeyManager) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if m.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := m.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if opErr == ErrInvalidArgument || opErr == ErrNotSupported {
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

// Encrypt runs Encrypt with resilience.
func (m *ResilientKeyManager) Encrypt(ctx context.Context, keyID string, plaintext []byte) ([]byte, error) {
	var out []byte
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		out, e = m.next.Encrypt(ctx, keyID, plaintext)
		return e
	})
	return out, err
}

// Decrypt runs Decrypt with resilience.
func (m *ResilientKeyManager) Decrypt(ctx context.Context, keyID string, ciphertext []byte) ([]byte, error) {
	var out []byte
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		out, e = m.next.Decrypt(ctx, keyID, ciphertext)
		return e
	})
	return out, err
}

// Unwrap returns the underlying key manager.
func (m *ResilientKeyManager) Unwrap() KeyManager {
	return m.next
}
