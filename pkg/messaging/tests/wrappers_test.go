package messaging_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
)

func TestNewFromConfigMemory(t *testing.T) {
	broker, err := messaging.NewFromConfig(messaging.Config{Driver: "memory", BufferSize: 8})
	if err != nil {
		t.Fatalf("NewFromConfig: %v", err)
	}
	defer broker.Close()

	if !broker.Healthy(context.Background()) {
		t.Fatal("expected healthy broker")
	}
}

func TestNewFromConfigUnregistered(t *testing.T) {
	_, err := messaging.NewFromConfig(messaging.Config{Driver: "kafka"})
	if err == nil {
		t.Fatal("expected error for unregistered kafka driver")
	}
	if !errors.IsCode(err, messaging.CodeInvalidConfig) {
		t.Fatalf("want CodeInvalidConfig, got %v", err)
	}
}

func TestPublishOptionsWired(t *testing.T) {
	msg := &messaging.Message{ID: "1", Payload: []byte("x")}
	messaging.ApplyPublishOptions(msg,
		messaging.WithDelay(5),
		messaging.WithOrderingKey("ord"),
		messaging.WithMessageGroupID("g1"),
		messaging.WithDeduplicationID("d1"),
	)
	opts := messaging.ParsePublishOptions(msg)
	if opts.DelaySeconds != 5 || opts.OrderingKey != "ord" || opts.MessageGroupID != "g1" || opts.DeduplicationID != "d1" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if string(msg.Key) != "ord" {
		t.Fatalf("expected Key set from ordering key, got %q", msg.Key)
	}
}

func TestConsumeOptionsContext(t *testing.T) {
	ctx := messaging.ContextWithConsumeOptions(context.Background(),
		messaging.WithMaxMessages(7),
		messaging.WithWaitTime(2*time.Second),
		messaging.WithVisibilityTimeout(30*time.Second),
	)
	opts, ok := messaging.ConsumeOptionsFromContext(ctx)
	if !ok {
		t.Fatal("expected consume options on context")
	}
	if opts.MaxMessages != 7 || opts.WaitTime != 2*time.Second || opts.VisibilityTimeout != 30*time.Second {
		t.Fatalf("unexpected consume opts: %+v", opts)
	}
}

func TestMemoryErrQueueFull(t *testing.T) {
	broker := memory.New(memory.Config{BufferSize: 1})
	defer broker.Close()

	producer, err := broker.Producer("full-topic")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := broker.Consumer("full-topic", "g")
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()

	ctx := context.Background()
	if err := producer.Publish(ctx, &messaging.Message{ID: "a", Payload: []byte("1")}); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	err = producer.Publish(ctx, &messaging.Message{ID: "b", Payload: []byte("2")})
	if err == nil {
		t.Fatal("expected ErrQueueFull on second publish")
	}
	if !errors.IsCode(err, messaging.CodeQueueFull) {
		t.Fatalf("want CodeQueueFull, got %v", err)
	}
}

func TestInstrumentedBrokerPublishConsume(t *testing.T) {
	inner := memory.New(memory.Config{BufferSize: 16})
	defer inner.Close()
	broker := messaging.NewInstrumentedBroker(inner)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	producer, err := broker.Producer("inst-topic")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := broker.Consumer("inst-topic", "inst-group")
	if err != nil {
		t.Fatal(err)
	}

	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			got = string(msg.Payload)
			wg.Done()
			cancel()
			return nil
		})
	}()

	if err := producer.Publish(ctx, &messaging.Message{ID: "m1", Payload: []byte("hello")}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestResilientConsumerRetriesHandler(t *testing.T) {
	inner := memory.New(memory.Config{BufferSize: 16})
	defer inner.Close()

	producer, err := inner.Producer("retry-topic")
	if err != nil {
		t.Fatal(err)
	}
	base, err := inner.Consumer("retry-topic", "retry-group")
	if err != nil {
		t.Fatal(err)
	}

	rc := messaging.NewResilientConsumer(base, messaging.ResilientBrokerConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      3,
		RetryBackoff:          time.Millisecond,
	})

	var attempts atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = rc.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			if attempts.Add(1) < 3 {
				return fmt.Errorf("transient")
			}
			wg.Done()
			cancel()
			return nil
		})
	}()

	if err := producer.Publish(ctx, &messaging.Message{ID: "r1", Payload: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestResilientBrokerProduceConsume(t *testing.T) {
	inner := memory.New(memory.Config{BufferSize: 16})
	defer inner.Close()
	broker := messaging.NewResilientBroker(inner, messaging.ResilientBrokerConfig{
		CircuitBreakerEnabled: false,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	producer, err := broker.Producer("rb-topic")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := broker.Consumer("rb-topic", "rb-group")
	if err != nil {
		t.Fatal(err)
	}

	var got string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			got = string(msg.Payload)
			wg.Done()
			cancel()
			return nil
		})
	}()

	if err := producer.Publish(ctx, &messaging.Message{ID: "m", Payload: []byte("ok")}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if got != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestDeduplicatingConsumerSkipsDuplicates(t *testing.T) {
	inner := memory.New(memory.Config{BufferSize: 16})
	defer inner.Close()

	producer, err := inner.Producer("dedup-topic")
	if err != nil {
		t.Fatal(err)
	}
	base, err := inner.Consumer("dedup-topic", "dedup-group")
	if err != nil {
		t.Fatal(err)
	}
	dc := messaging.NewDeduplicatingConsumer(base, messaging.DeduplicationConfig{
		ExpectedMessages:  100,
		FalsePositiveRate: 0.01,
	})

	var handled atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = dc.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			handled.Add(1)
			select {
			case <-done:
			default:
				close(done)
			}
			return nil
		})
	}()

	if err := producer.Publish(ctx, &messaging.Message{ID: "same-id", Payload: []byte("once")}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first message")
	}

	if err := producer.Publish(ctx, &messaging.Message{ID: "same-id", Payload: []byte("once")}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if handled.Load() != 1 {
		t.Fatalf("expected exactly 1 handle, got %d", handled.Load())
	}
}

func TestDeduplicatingConsumerTOCTOU(t *testing.T) {
	msg := &messaging.Message{ID: "race-id", Payload: []byte("x")}
	fan := &fanConsumer{msg: msg, n: 2}
	dc := messaging.NewDeduplicatingConsumer(fan, messaging.DeduplicationConfig{
		ExpectedMessages:  64,
		FalsePositiveRate: 0.01,
	})

	var handled atomic.Int32
	if err := dc.Consume(context.Background(), func(ctx context.Context, m *messaging.Message) error {
		time.Sleep(20 * time.Millisecond)
		handled.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if handled.Load() != 1 {
		t.Fatalf("TOCTOU: expected 1 handler success, got %d", handled.Load())
	}
}

// fanConsumer delivers the same message to handler n times concurrently, then returns.
type fanConsumer struct {
	msg *messaging.Message
	n   int
}

func (f *fanConsumer) Consume(ctx context.Context, handler messaging.MessageHandler) error {
	var wg sync.WaitGroup
	wg.Add(f.n)
	for i := 0; i < f.n; i++ {
		go func() {
			defer wg.Done()
			_ = handler(ctx, f.msg)
		}()
	}
	wg.Wait()
	return nil
}

func (f *fanConsumer) Close() error { return nil }

func TestPublishHelper(t *testing.T) {
	broker := memory.New(memory.Config{BufferSize: 8})
	defer broker.Close()
	producer, err := broker.Producer("opt-topic")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := broker.Consumer("opt-topic", "g")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var received *messaging.Message
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		_ = messaging.Consume(ctx, consumer, func(ctx context.Context, msg *messaging.Message) error {
			received = msg
			wg.Done()
			cancel()
			return nil
		}, messaging.WithMaxMessages(1))
	}()

	msg := &messaging.Message{ID: "p1", Payload: []byte("body")}
	if err := messaging.Publish(ctx, producer, msg, messaging.WithOrderingKey("k1")); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if received == nil {
		t.Fatal("no message")
	}
	opts := messaging.ParsePublishOptions(received)
	if opts.OrderingKey != "k1" {
		t.Fatalf("ordering key not preserved: %+v", opts)
	}
}
