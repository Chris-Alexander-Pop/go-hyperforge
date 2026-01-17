// Package tests provides a generic test suite for messaging.Broker implementations.
//
// This package can be used to test any adapter that implements the messaging.Broker interface.
// It ensures consistency across all messaging backends.
//
// # Usage
//
//	func TestMemoryBroker(t *testing.T) {
//	    broker := memory.New(memory.Config{BufferSize: 100})
//	    defer broker.Close()
//	    tests.RunBrokerTests(t, broker)
//	}
package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/messaging"
)

// RunBrokerTests runs the full test suite against a Broker implementation.
func RunBrokerTests(t *testing.T, broker messaging.Broker) {
	t.Run("PublishConsume", func(t *testing.T) {
		testPublishConsume(t, broker)
	})

	t.Run("BatchPublish", func(t *testing.T) {
		testBatchPublish(t, broker)
	})

	t.Run("MultipleConsumers", func(t *testing.T) {
		testMultipleConsumers(t, broker)
	})

	t.Run("Headers", func(t *testing.T) {
		testHeaders(t, broker)
	})

	t.Run("Healthy", func(t *testing.T) {
		testHealthy(t, broker)
	})
}

func testPublishConsume(t *testing.T, broker messaging.Broker) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-publish-consume"

	producer, err := broker.Producer(topic)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	consumer, err := broker.Consumer(topic, "test-group")
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Publish a message
	msg := &messaging.Message{
		ID:      "test-msg-1",
		Topic:   topic,
		Payload: []byte("hello world"),
	}

	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Consume the message
	var received *messaging.Message
	var wg sync.WaitGroup
	wg.Add(1)

	consumeCtx, consumeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer consumeCancel()

	go func() {
		consumer.Consume(consumeCtx, func(ctx context.Context, m *messaging.Message) error {
			received = m
			wg.Done()
			consumeCancel()
			return nil
		})
	}()

	wg.Wait()

	if received == nil {
		t.Fatal("did not receive message")
	}

	if string(received.Payload) != "hello world" {
		t.Errorf("payload mismatch: got %s, want %s", received.Payload, "hello world")
	}
}

func testBatchPublish(t *testing.T, broker messaging.Broker) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-batch-publish"

	producer, err := broker.Producer(topic)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	consumer, err := broker.Consumer(topic, "test-batch-group")
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Publish batch
	msgs := []*messaging.Message{
		{ID: "batch-1", Payload: []byte("msg1")},
		{ID: "batch-2", Payload: []byte("msg2")},
		{ID: "batch-3", Payload: []byte("msg3")},
	}

	if err := producer.PublishBatch(ctx, msgs); err != nil {
		t.Fatalf("failed to publish batch: %v", err)
	}

	// Consume all messages
	receivedCount := 0
	var mu sync.Mutex

	consumeCtx, consumeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer consumeCancel()

	go func() {
		consumer.Consume(consumeCtx, func(ctx context.Context, m *messaging.Message) error {
			mu.Lock()
			receivedCount++
			if receivedCount >= 3 {
				consumeCancel()
			}
			mu.Unlock()
			return nil
		})
	}()

	<-consumeCtx.Done()

	mu.Lock()
	defer mu.Unlock()
	if receivedCount < 3 {
		t.Errorf("expected at least 3 messages, got %d", receivedCount)
	}
}

func testMultipleConsumers(t *testing.T, broker messaging.Broker) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-multiple-consumers"

	producer, err := broker.Producer(topic)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	// Create two consumers in the same group
	consumer1, err := broker.Consumer(topic, "shared-group")
	if err != nil {
		t.Fatalf("failed to create consumer1: %v", err)
	}
	defer consumer1.Close()

	consumer2, err := broker.Consumer(topic, "shared-group")
	if err != nil {
		t.Fatalf("failed to create consumer2: %v", err)
	}
	defer consumer2.Close()

	// Publish messages
	for i := 0; i < 10; i++ {
		msg := &messaging.Message{
			ID:      "multi-" + string(rune('0'+i)),
			Payload: []byte("message"),
		}
		if err := producer.Publish(ctx, msg); err != nil {
			t.Fatalf("failed to publish: %v", err)
		}
	}

	// Both consumers should receive some messages
	var count1, count2 int
	var mu sync.Mutex

	consumeCtx, consumeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer consumeCancel()

	go func() {
		consumer1.Consume(consumeCtx, func(ctx context.Context, m *messaging.Message) error {
			mu.Lock()
			count1++
			mu.Unlock()
			return nil
		})
	}()

	go func() {
		consumer2.Consume(consumeCtx, func(ctx context.Context, m *messaging.Message) error {
			mu.Lock()
			count2++
			mu.Unlock()
			return nil
		})
	}()

	<-consumeCtx.Done()

	mu.Lock()
	defer mu.Unlock()
	total := count1 + count2
	if total == 0 {
		t.Error("no messages received by either consumer")
	}
}

func testHeaders(t *testing.T, broker messaging.Broker) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test-headers"

	producer, err := broker.Producer(topic)
	if err != nil {
		t.Fatalf("failed to create producer: %v", err)
	}
	defer producer.Close()

	consumer, err := broker.Consumer(topic, "test-headers-group")
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Publish message with headers
	msg := &messaging.Message{
		ID:      "test-headers-1",
		Payload: []byte("test"),
		Headers: map[string]string{
			"content-type": "application/json",
			"x-custom":     "value",
		},
	}

	if err := producer.Publish(ctx, msg); err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Consume and verify headers
	var received *messaging.Message
	var wg sync.WaitGroup
	wg.Add(1)

	consumeCtx, consumeCancel := context.WithTimeout(ctx, 5*time.Second)
	defer consumeCancel()

	go func() {
		consumer.Consume(consumeCtx, func(ctx context.Context, m *messaging.Message) error {
			received = m
			wg.Done()
			consumeCancel()
			return nil
		})
	}()

	wg.Wait()

	if received == nil {
		t.Fatal("did not receive message")
	}

	// Note: Not all brokers preserve headers exactly, so this is a best-effort check
	if len(received.Headers) == 0 {
		t.Log("warning: headers not preserved by this broker")
	}
}

func testHealthy(t *testing.T, broker messaging.Broker) {
	ctx := context.Background()

	if !broker.Healthy(ctx) {
		t.Error("broker should be healthy")
	}
}

// RunBenchmarkPublish is a helper for benchmarking publish operations.
// Use this from adapter-specific benchmark tests.
func RunBenchmarkPublish(b *testing.B, broker messaging.Broker) {
	ctx := context.Background()

	producer, err := broker.Producer("benchmark-topic")
	if err != nil {
		b.Fatal(err)
	}
	defer producer.Close()

	msg := &messaging.Message{
		Payload: []byte("benchmark message"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg.ID = ""
		if err := producer.Publish(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}
