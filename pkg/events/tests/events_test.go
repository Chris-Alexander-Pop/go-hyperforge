package events_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

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
