package messaging

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
)

// ResilientBrokerConfig configures the resilient broker wrapper.
type ResilientBrokerConfig struct {
	// Circuit breaker settings
	CircuitBreakerEnabled   bool          `env:"MSG_CB_ENABLED" env-default:"true"`
	CircuitBreakerThreshold int64         `env:"MSG_CB_THRESHOLD" env-default:"5"`
	CircuitBreakerTimeout   time.Duration `env:"MSG_CB_TIMEOUT" env-default:"30s"`

	// Retry settings
	RetryEnabled     bool          `env:"MSG_RETRY_ENABLED" env-default:"true"`
	RetryMaxAttempts int           `env:"MSG_RETRY_MAX" env-default:"3"`
	RetryBackoff     time.Duration `env:"MSG_RETRY_BACKOFF" env-default:"100ms"`
}

// ResilientBroker wraps a Broker with circuit breaker and retry support.
type ResilientBroker struct {
	broker   Broker
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientBroker wraps a broker with resilience features.
func NewResilientBroker(broker Broker, cfg ResilientBrokerConfig) *ResilientBroker {
	rb := &ResilientBroker{
		broker: broker,
	}

	if cfg.CircuitBreakerEnabled {
		rb.cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             "messaging",
			FailureThreshold: cfg.CircuitBreakerThreshold,
			SuccessThreshold: 2,
			Timeout:          cfg.CircuitBreakerTimeout,
		})
	}

	if cfg.RetryEnabled {
		rb.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: cfg.RetryBackoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
		}
	}

	return rb
}

func (rb *ResilientBroker) Producer(topic string) (Producer, error) {
	var producer Producer
	err := rb.execute(context.Background(), func(ctx context.Context) error {
		var err error
		producer, err = rb.broker.Producer(topic)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &resilientProducer{
		producer: producer,
		broker:   rb,
	}, nil
}

func (rb *ResilientBroker) Consumer(topic string, group string) (Consumer, error) {
	return rb.broker.Consumer(topic, group)
}

func (rb *ResilientBroker) Close() error {
	return rb.broker.Close()
}

func (rb *ResilientBroker) Healthy(ctx context.Context) bool {
	return rb.broker.Healthy(ctx)
}

func (rb *ResilientBroker) execute(ctx context.Context, fn resilience.Executor) error {
	operation := fn

	if rb.cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			return rb.cb.Execute(ctx, cbFn)
		}
	}

	if rb.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, rb.retryCfg, operation)
	}

	return operation(ctx)
}

// resilientProducer wraps a producer with resilience.
type resilientProducer struct {
	producer Producer
	broker   *ResilientBroker
}

func (rp *resilientProducer) Publish(ctx context.Context, msg *Message) error {
	return rp.broker.execute(ctx, func(ctx context.Context) error {
		return rp.producer.Publish(ctx, msg)
	})
}

func (rp *resilientProducer) PublishBatch(ctx context.Context, msgs []*Message) error {
	return rp.broker.execute(ctx, func(ctx context.Context) error {
		return rp.producer.PublishBatch(ctx, msgs)
	})
}

func (rp *resilientProducer) Close() error {
	return rp.producer.Close()
}
