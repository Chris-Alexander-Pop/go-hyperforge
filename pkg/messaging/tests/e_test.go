package messaging_test

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
)

func Example() {
	// Create an in-memory message broker
	broker := memory.New(memory.Config{BufferSize: 100})
	defer broker.Close()

	ctx := context.Background()

	// Create a producer
	producer, _ := broker.Producer("orders")

	// Create a consumer
	consumer, _ := broker.Consumer("orders", "order-processor")

	// Start consuming in background
	done := make(chan struct{})
	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			fmt.Printf("Received: %s\n", string(msg.Payload))
			close(done)
			return nil
		})
	}()

	// Publish a message
	msg := &messaging.Message{
		ID:      "msg-123",
		Payload: []byte(`{"order_id": "123", "status": "created"}`),
	}

	_ = producer.Publish(ctx, msg)

	// Wait for message to be processed
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
	}
	// Output: Received: {"order_id": "123", "status": "created"}
}
