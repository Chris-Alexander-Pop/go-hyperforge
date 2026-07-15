package iot

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure ResilientClient implements Client at compile time.
var _ Client = (*ResilientClient)(nil)

// ResilientConfig configures the resilient IoT MQTT client wrapper.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

// ResilientClient wraps a Client with circuit breaker and retry.
// Invalid-config / not-connected outcomes are not retried and do not open the circuit.
type ResilientClient struct {
	next     Client
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientClient wraps next with resilience features.
func NewResilientClient(next Client, cfg ResilientConfig) *ResilientClient {
	rc := &ResilientClient{next: next}

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		rc.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "iot",
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
		rc.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				if errors.IsCode(err, CodeInvalidConfig) || errors.IsCode(err, CodeNotConnected) {
					return false
				}
				return true
			},
		}
	}

	return rc
}

func (c *ResilientClient) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn
	if c.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := c.cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if errors.IsCode(opErr, CodeInvalidConfig) || errors.IsCode(opErr, CodeNotConnected) {
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
	if c.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, c.retryCfg, operation)
	}
	return operation(ctx)
}

// Connect runs Connect with resilience.
func (c *ResilientClient) Connect(ctx context.Context) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.Connect(ctx)
	})
}

// Disconnect delegates without resilience (local state change).
func (c *ResilientClient) Disconnect() {
	c.next.Disconnect()
}

// IsConnected delegates without resilience.
func (c *ResilientClient) IsConnected() bool {
	return c.next.IsConnected()
}

// Publish runs Publish with resilience.
func (c *ResilientClient) Publish(ctx context.Context, topic string, payload []byte) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.Publish(ctx, topic, payload)
	})
}

// PublishWithOptions runs PublishWithOptions with resilience.
func (c *ResilientClient) PublishWithOptions(ctx context.Context, topic string, payload []byte, qos QoS, retained bool) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.PublishWithOptions(ctx, topic, payload, qos, retained)
	})
}

// Subscribe runs Subscribe with resilience.
func (c *ResilientClient) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.Subscribe(ctx, topic, handler)
	})
}

// SubscribeWithQoS runs SubscribeWithQoS with resilience.
func (c *ResilientClient) SubscribeWithQoS(ctx context.Context, topic string, qos QoS, handler MessageHandler) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.SubscribeWithQoS(ctx, topic, qos, handler)
	})
}

// Unsubscribe runs Unsubscribe with resilience.
func (c *ResilientClient) Unsubscribe(ctx context.Context, topic string) error {
	return c.execute(ctx, func(ctx context.Context) error {
		return c.next.Unsubscribe(ctx, topic)
	})
}

// Unwrap returns the underlying client.
func (c *ResilientClient) Unwrap() Client {
	return c.next
}
