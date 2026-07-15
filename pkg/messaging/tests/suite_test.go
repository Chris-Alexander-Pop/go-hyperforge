package messaging_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
)

// WrappersSuite migrates messaging wrapper smoke tests onto pkg/test.Suite.
type WrappersSuite struct {
	test.Suite
}

func (s *WrappersSuite) TestNewFromConfigMemory() {
	broker, err := messaging.NewFromConfig(messaging.Config{Driver: "memory", BufferSize: 8})
	s.NoError(err)
	defer broker.Close()
	s.True(broker.Healthy(s.Ctx))
}

func (s *WrappersSuite) TestNewFromConfigUnregistered() {
	_, err := messaging.NewFromConfig(messaging.Config{Driver: "kafka"})
	s.Error(err)
	s.True(errors.IsCode(err, messaging.CodeInvalidConfig))
}

func (s *WrappersSuite) TestMemoryErrQueueFull() {
	broker := memory.New(memory.Config{BufferSize: 1})
	defer broker.Close()

	producer, err := broker.Producer("full-topic")
	s.NoError(err)
	consumer, err := broker.Consumer("full-topic", "g")
	s.NoError(err)
	defer consumer.Close()

	s.NoError(producer.Publish(s.Ctx, &messaging.Message{ID: "a", Payload: []byte("1")}))
	err = producer.Publish(s.Ctx, &messaging.Message{ID: "b", Payload: []byte("2")})
	s.Error(err)
	s.True(errors.IsCode(err, messaging.CodeQueueFull))
}

func (s *WrappersSuite) TestResilientConsumerRetriesHandler() {
	inner := memory.New(memory.Config{BufferSize: 16})
	defer inner.Close()

	producer, err := inner.Producer("retry-topic")
	s.NoError(err)
	base, err := inner.Consumer("retry-topic", "retry-group")
	s.NoError(err)

	rc := messaging.NewResilientConsumer(base, messaging.ResilientBrokerConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      3,
		RetryBackoff:          time.Millisecond,
	})

	var attempts atomic.Int32
	ctx, cancel := context.WithTimeout(s.Ctx, 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = rc.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			if attempts.Add(1) < 3 {
				return fmt.Errorf("transient")
			}
			close(done)
			cancel()
			return nil
		})
	}()

	s.NoError(producer.Publish(ctx, &messaging.Message{ID: "r1", Payload: []byte("x")}))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.Fail("timed out waiting for resilient consumer")
	}
	s.Equal(int32(3), attempts.Load())
}

func TestWrappersSuite(t *testing.T) {
	test.Run(t, new(WrappersSuite))
}
