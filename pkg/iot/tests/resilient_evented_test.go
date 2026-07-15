package iot_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/iot"
	iotmem "github.com/chris-alexander-pop/go-hyperforge/pkg/iot/adapters/memory"
)

func TestResilientClient_Publish(t *testing.T) {
	inner := iotmem.NewClient()
	client := iot.NewResilientClient(inner, iot.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(ctx, "t/a", []byte("hi")); err != nil {
		t.Fatal(err)
	}
}

func TestEventedClient_Publish(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var got events.Event
	done := make(chan struct{}, 1)
	_, err := bus.Subscribe(context.Background(), iot.TopicIoT, func(ctx context.Context, ev events.Event) error {
		got = ev
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	inner := iotmem.NewClient()
	client := iot.NewEventedClient(inner, bus)
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(ctx, "devices/1", []byte("ping")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	if got.Type != iot.EventTypePublished {
		t.Fatalf("type=%s", got.Type)
	}
	payload, ok := got.Payload.(iot.MessageEventPayload)
	if !ok {
		t.Fatalf("payload %T", got.Payload)
	}
	if payload.Topic != "devices/1" {
		t.Fatalf("topic=%s", payload.Topic)
	}
}

func TestEventedClient_NilBus(t *testing.T) {
	client := iot.NewEventedClient(iotmem.NewClient(), nil)
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := client.Publish(ctx, "t", []byte("x")); err != nil {
		t.Fatal(err)
	}
}
