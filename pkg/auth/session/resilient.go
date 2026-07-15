package session

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientManager implements Manager at compile time.
var _ Manager = (*ResilientManager)(nil)

// ResilientConfig configures the resilient session manager wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientManager wraps a Manager with circuit breaker and retry.
// Session-not-found outcomes are not retried and do not open the circuit.
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
			Name:             "auth-session",
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
		rm.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				if errors.Is(err, auth.ErrSessionNotFound) {
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
				if errors.Is(opErr, auth.ErrSessionNotFound) {
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

// Create runs Create with resilience.
func (m *ResilientManager) Create(ctx context.Context, userID string, metadata map[string]interface{}) (*Session, error) {
	var s *Session
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		s, e = m.next.Create(ctx, userID, metadata)
		return e
	})
	return s, err
}

// Get runs Get with resilience.
func (m *ResilientManager) Get(ctx context.Context, sessionID string) (*Session, error) {
	var s *Session
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		s, e = m.next.Get(ctx, sessionID)
		return e
	})
	return s, err
}

// Delete runs Delete with resilience.
func (m *ResilientManager) Delete(ctx context.Context, sessionID string) error {
	return m.execute(ctx, func(ctx context.Context) error {
		return m.next.Delete(ctx, sessionID)
	})
}

// Refresh runs Refresh with resilience.
func (m *ResilientManager) Refresh(ctx context.Context, sessionID string) (*Session, error) {
	var s *Session
	err := m.execute(ctx, func(ctx context.Context) error {
		var e error
		s, e = m.next.Refresh(ctx, sessionID)
		return e
	})
	return s, err
}

// Unwrap returns the underlying manager.
func (m *ResilientManager) Unwrap() Manager {
	return m.next
}
