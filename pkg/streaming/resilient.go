package streaming

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// ResilientConfig configures the resilient streaming client wrapper.
type ResilientConfig struct {
	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `env:"STREAMING_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"STREAMING_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"STREAMING_CB_TIMEOUT" env-default:"30s"`

	// Retry settings
	RetryEnabled     bool          `env:"STREAMING_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"STREAMING_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"STREAMING_RETRY_BACKOFF" env-default:"100ms"`
}

// ResilientClient wraps a Client with circuit breaker and retry support.
type ResilientClient struct {
	client   Client
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// Ensure ResilientClient implements Client.
var _ Client = (*ResilientClient)(nil)

// NewResilientClient wraps a client with resilience features.
func NewResilientClient(client Client, cfg ResilientConfig) *ResilientClient {
	rc := &ResilientClient{
		client: client,
	}

	if cfg.CircuitBreakerEnabled {
		rc.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "streaming",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled {
		rc.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			RetryIf: func(err error) bool {
				// Closed / buffer-full are not transient.
				return err != nil && !IsClosed(err) && !IsBufferFull(err)
			},
		}
	}

	return rc
}

func (rc *ResilientClient) PutRecord(ctx context.Context, streamName string, partitionKey string, data []byte) error {
	return rc.execute(ctx, func(ctx context.Context) error {
		return rc.client.PutRecord(ctx, streamName, partitionKey, data)
	})
}

func (rc *ResilientClient) PutRecords(ctx context.Context, records []Record) error {
	return rc.execute(ctx, func(ctx context.Context) error {
		return rc.client.PutRecords(ctx, records)
	})
}

func (rc *ResilientClient) Close() error {
	return rc.client.Close()
}

func (rc *ResilientClient) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	if rc.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return rc.cb.Execute(ctx, cbFn)
		}
	}

	if rc.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, rc.retryCfg, operation)
	}

	return operation(ctx)
}

// Unwrap returns the underlying client.
func (rc *ResilientClient) Unwrap() Client {
	return rc.client
}

// CircuitBreakerState returns the current circuit breaker state.
func (rc *ResilientClient) CircuitBreakerState() resilience.State {
	if rc.cb == nil {
		return ""
	}
	return rc.cb.State()
}
