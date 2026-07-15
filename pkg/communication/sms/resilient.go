package sms

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/communication"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// ResilientSender wraps a Sender with retry (and optional circuit breaker) support.
type ResilientSender struct {
	next     Sender
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// ResilientConfig configures the resilient SMS sender wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool          `env:"SMS_CB_ENABLED" env-default:"false"`
	CircuitBreakerThreshold int64         `env:"SMS_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"SMS_CB_TIMEOUT" env-default:"30s"`

	RetryEnabled     bool          `env:"SMS_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"SMS_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"SMS_RETRY_BACKOFF" env-default:"1s"`
}

// NewResilientSender wraps a sender with resilience features.
func NewResilientSender(next Sender, cfg ResilientConfig) *ResilientSender {
	rs := &ResilientSender{next: next}

	if cfg.CircuitBreakerEnabled {
		rs.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "sms",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		rs.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     30 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf:        communication.ShouldRetrySend,
		}
	}

	return rs
}

// ResilientConfigFrom maps sms.Config retry fields into a ResilientConfig.
func ResilientConfigFrom(cfg Config) ResilientConfig {
	return ResilientConfig{
		RetryEnabled:     cfg.RetryMax > 0,
		RetryMaxAttempts: cfg.RetryMax,
		RetryBackoff:     cfg.RetryBackoff,
	}
}

// Send dispatches a single SMS with retry/circuit-breaker protection.
func (s *ResilientSender) Send(ctx context.Context, msg *Message) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.Send(ctx, msg)
	})
}

// SendBatch dispatches multiple SMS messages with retry/circuit-breaker protection.
func (s *ResilientSender) SendBatch(ctx context.Context, msgs []*Message) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.SendBatch(ctx, msgs)
	})
}

// Close releases resources held by the underlying sender.
func (s *ResilientSender) Close() error {
	return s.next.Close()
}

// Unwrap returns the underlying sender.
func (s *ResilientSender) Unwrap() Sender {
	return s.next
}

func (s *ResilientSender) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	if s.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return s.cb.Execute(ctx, cbFn)
		}
	}

	if s.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, s.retryCfg, operation)
	}

	return operation(ctx)
}

var _ Sender = (*ResilientSender)(nil)
