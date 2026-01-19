package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/events/adapters/memory"
)

func TestMemoryBus(t *testing.T) {
	bus := memory.New()
	defer bus.Close()

	ctx := context.Background()
	topic := "test.topic"

	var wg sync.WaitGroup
	wg.Add(1)

	var received events.Event

	err := bus.Subscribe(ctx, topic, func(ctx context.Context, e events.Event) error {
		received = e
		wg.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	payload := map[string]string{"foo": "bar"}
	evt := events.Event{
		ID:        "123",
		Type:      "test.event",
		Source:    "test",
		Timestamp: time.Now(),
		Payload:   payload,
	}

	err = bus.Publish(ctx, topic, evt)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for async handler
	// Note: MemoryBus implementation might be sync or async.
	// If sync, WG is done immediately. If async, we wait.
	// Looking at previous grep, it launched a goroutine.
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for event handler")
	}

	if received.ID != "123" {
		t.Errorf("Expected event ID 123, got %s", received.ID)
	}
}
