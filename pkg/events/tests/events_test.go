package events_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	"github.com/chris-alexander-pop/system-design-library/pkg/events/adapters/memory"
)

func TestMemoryBusPublishSubscribe(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	topic := "users"

	var received events.Event
	_, err := bus.Subscribe(ctx, topic, func(ctx context.Context, e events.Event) error {
		received = e
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	evt := events.Event{
		ID:      "123",
		Type:    "user.created",
		Source:  "test",
		Payload: map[string]string{"foo": "bar"},
	}

	if err := bus.Publish(ctx, topic, evt); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	if received.ID != "123" {
		t.Errorf("Expected event ID 123, got %s", received.ID)
	}
	if received.Timestamp.IsZero() {
		t.Error("Expected Publish to set Timestamp when zero")
	}
}

func TestMemoryBusMultiSubscriber(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	var count atomic.Int32

	for i := 0; i < 3; i++ {
		_, err := bus.Subscribe(ctx, "orders", func(ctx context.Context, e events.Event) error {
			count.Add(1)
			return nil
		})
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
	}

	err := bus.Publish(ctx, "orders", events.Event{Type: "order.placed", ID: "o1"})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if got := count.Load(); got != 3 {
		t.Fatalf("expected 3 handlers, got %d", got)
	}
}

func TestMemoryBusUnsubscribe(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	var count atomic.Int32

	sub, err := bus.Subscribe(ctx, "payments", func(ctx context.Context, e events.Event) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Publish(ctx, "payments", events.Event{Type: "payment.captured"}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if count.Load() != 1 {
		t.Fatalf("expected 1 delivery before unsubscribe")
	}

	if err := bus.Unsubscribe(ctx, sub); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	if err := bus.Publish(ctx, "payments", events.Event{Type: "payment.captured"}); err != nil {
		t.Fatalf("Publish after unsubscribe failed: %v", err)
	}
	if count.Load() != 1 {
		t.Fatalf("expected no further deliveries after unsubscribe, got %d", count.Load())
	}

	err = bus.Unsubscribe(ctx, sub)
	if err == nil {
		t.Fatal("expected ErrSubscriptionNotFound")
	}
	var appErr *errors.AppError
	if !errors.As(err, &appErr) || appErr.Code != events.CodeSubscriptionNotFound {
		t.Fatalf("expected subscription not found, got %v", err)
	}
}

func TestMemoryBusHandlerError(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	_, err := bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
		return fmt.Errorf("boom")
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = bus.Publish(ctx, "users", events.Event{Type: "user.updated"})
	if err == nil {
		t.Fatal("expected handler error from Publish")
	}
	var appErr *errors.AppError
	if !errors.As(err, &appErr) || appErr.Code != events.CodeHandlerFailed {
		t.Fatalf("expected EVENTS_HANDLER_FAILED, got %v", err)
	}
}

func TestMemoryBusClose(t *testing.T) {
	bus := memory.New(events.Config{})
	ctx := context.Background()

	_, err := bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := bus.Close(); err != nil {
		t.Fatalf("second Close should be idempotent: %v", err)
	}

	err = bus.Publish(ctx, "users", events.Event{Type: "user.created"})
	if err == nil {
		t.Fatal("expected ErrClosed on Publish")
	}
	var appErr *errors.AppError
	if !errors.As(err, &appErr) || appErr.Code != events.CodeClosed {
		t.Fatalf("expected EVENTS_CLOSED, got %v", err)
	}

	_, err = bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error { return nil })
	if err == nil {
		t.Fatal("expected ErrClosed on Subscribe")
	}
}

func TestMemoryBusInvalidEvent(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	err := bus.Publish(ctx, "users", events.Event{})
	if err == nil {
		t.Fatal("expected invalid event error")
	}
	var appErr *errors.AppError
	if !errors.As(err, &appErr) || appErr.Code != events.CodeInvalidEvent {
		t.Fatalf("expected EVENTS_INVALID_EVENT, got %v", err)
	}

	err = bus.Publish(ctx, "", events.Event{Type: "user.created"})
	if err == nil {
		t.Fatal("expected invalid topic error")
	}
	if !errors.As(err, &appErr) || appErr.Code != events.CodeInvalidTopic {
		t.Fatalf("expected EVENTS_INVALID_TOPIC, got %v", err)
	}
}

func TestMemoryBusPropagatesContext(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "trace-1")

	var got any
	_, err := bus.Subscribe(ctx, "users", func(handlerCtx context.Context, e events.Event) error {
		got = handlerCtx.Value(key{})
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Publish(ctx, "users", events.Event{Type: "user.created"}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if got != "trace-1" {
		t.Fatalf("expected publish ctx value in handler, got %v", got)
	}
}

func TestMemoryBusAsyncCloseWaits(t *testing.T) {
	bus := memory.New(events.Config{Async: true, Workers: 2, QueueSize: 8})

	ctx := context.Background()
	started := make(chan struct{})
	release := make(chan struct{})

	_, err := bus.Subscribe(ctx, "notifications", func(ctx context.Context, e events.Event) error {
		close(started)
		<-release
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Publish(ctx, "notifications", events.Event{Type: "notification.sent"}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}

	done := make(chan struct{})
	go func() {
		_ = bus.Close()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Close returned before in-flight handler finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not complete after handler finished")
	}
}

func TestMemoryBusRacePublishSubscribe(t *testing.T) {
	bus := memory.New(events.Config{})
	defer bus.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	var deliveries atomic.Int64

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
				deliveries.Add(1)
				return nil
			})
		}()
	}

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = bus.Publish(ctx, "users", events.Event{
				ID:   fmt.Sprintf("%d", n),
				Type: "user.updated",
			})
		}(i)
	}

	wg.Wait()
	// Race-safe completion is the assertion; deliveries may be partial due to timing.
	_ = deliveries.Load()
}

func TestInstrumentedBusSmoke(t *testing.T) {
	inner := memory.New(events.Config{})
	bus := events.NewInstrumentedBus(inner)
	defer bus.Close()

	ctx := context.Background()
	var got string
	sub, err := bus.Subscribe(ctx, "users", func(ctx context.Context, e events.Event) error {
		got = e.Type
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if err := bus.Publish(ctx, "users", events.Event{Type: "user.created", ID: "1"}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if got != "user.created" {
		t.Fatalf("expected user.created, got %s", got)
	}

	if err := bus.Unsubscribe(ctx, sub); err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}
}
