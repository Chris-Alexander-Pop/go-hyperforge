package events_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	msgmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
)

func TestOutboxPublish(t *testing.T) {
	broker := msgmemory.New(msgmemory.Config{BufferSize: 16})
	defer broker.Close()

	producer, err := broker.Producer("users")
	if err != nil {
		t.Fatalf("Producer: %v", err)
	}
	consumer, err := broker.Consumer("users", "g1")
	if err != nil {
		t.Fatalf("Consumer: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	got := make(chan []byte, 1)
	go func() {
		_ = consumer.Consume(ctx, func(ctx context.Context, msg *messaging.Message) error {
			got <- msg.Payload
			cancel()
			return nil
		})
	}()

	outbox := events.NewOutbox(producer)
	err = outbox.Publish(context.Background(), "users", events.Event{
		Type:    "user.created",
		Source:  "user-service",
		Payload: map[string]string{"id": "u1"},
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case payload := <-got:
		var env events.OutboxPayload
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != "user.created" || env.Topic != "users" {
			t.Fatalf("envelope=%+v", env)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for outbox message")
	}
}

func TestOutboxBus(t *testing.T) {
	broker := msgmemory.New(msgmemory.Config{BufferSize: 8})
	defer broker.Close()
	producer, err := broker.Producer("orders")
	if err != nil {
		t.Fatalf("Producer: %v", err)
	}

	bus := eventsmemory.New(events.Config{})
	defer bus.Close()

	localHit := make(chan struct{}, 1)
	_, err = bus.Subscribe(context.Background(), "orders", func(ctx context.Context, e events.Event) error {
		localHit <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	ob := events.NewOutboxBus(bus, events.NewOutbox(producer))
	if err := ob.Publish(context.Background(), "orders", events.Event{
		Type:    "order.placed",
		Source:  "commerce",
		Payload: map[string]int{"n": 1},
	}); err != nil {
		t.Fatalf("OutboxBus.Publish: %v", err)
	}

	select {
	case <-localHit:
	case <-time.After(time.Second):
		t.Fatal("local bus did not receive event")
	}
}
