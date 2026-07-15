package payment

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientProvider implements Provider at compile time.
var _ Provider = (*ResilientProvider)(nil)

// ResilientProvider wraps a Provider with retry (and optional circuit breaker).
type ResilientProvider struct {
	next     Provider
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// ResilientConfig configures the resilient payment wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientConfigFrom maps payment.Config retry fields into a ResilientConfig.
func ResilientConfigFrom(cfg Config) ResilientConfig {
	return ResilientConfig{
		RetryEnabled:     cfg.RetryMaxAttempts > 0,
		RetryMaxAttempts: cfg.RetryMaxAttempts,
		RetryBackoff:     cfg.RetryBackoff,
	}
}

// NewResilientProvider wraps a provider with resilience features.
func NewResilientProvider(next Provider, cfg ResilientConfig) *ResilientProvider {
	rp := &ResilientProvider{next: next}

	if cfg.CircuitBreakerEnabled {
		rp.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "payment",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		rp.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				// Do not retry business declines / invalid input.
				switch err {
				case ErrDeclined, ErrInsufficientFunds, ErrInvalidCard, ErrExpiredCard,
					ErrInvalidWebhook, ErrNotAuthorized, ErrIdempotencyConflict:
					return false
				}
				return true
			},
		}
	}

	return rp
}

func (p *ResilientProvider) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if p.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return p.cb.Execute(ctx, cbFn)
		}
	}
	if p.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, p.retryCfg, operation)
	}
	return operation(ctx)
}

// Charge runs Charge with resilience.
func (p *ResilientProvider) Charge(ctx context.Context, req *ChargeRequest) (*Transaction, error) {
	var tx *Transaction
	err := p.execute(ctx, func(ctx context.Context) error {
		var e error
		tx, e = p.next.Charge(ctx, req)
		return e
	})
	return tx, err
}

// Refund runs Refund with resilience.
func (p *ResilientProvider) Refund(ctx context.Context, req *RefundRequest) (*Transaction, error) {
	var tx *Transaction
	err := p.execute(ctx, func(ctx context.Context) error {
		var e error
		tx, e = p.next.Refund(ctx, req)
		return e
	})
	return tx, err
}

// GetTransaction runs GetTransaction with resilience.
func (p *ResilientProvider) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	var tx *Transaction
	err := p.execute(ctx, func(ctx context.Context) error {
		var e error
		tx, e = p.next.GetTransaction(ctx, id)
		return e
	})
	return tx, err
}

// Close releases resources held by the underlying provider.
func (p *ResilientProvider) Close() error {
	return p.next.Close()
}

// Unwrap returns the underlying provider.
func (p *ResilientProvider) Unwrap() Provider {
	return p.next
}
