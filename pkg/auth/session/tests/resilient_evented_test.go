package tests

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session"
	sessionmem "github.com/chris-alexander-pop/go-hyperforge/pkg/auth/session/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestResilientManager_CreateGet(t *testing.T) {
	inner, err := sessionmem.New(session.Config{TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	mgr := session.NewResilientManager(inner, session.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	ctx := context.Background()
	s, err := mgr.Create(ctx, "u1", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := mgr.Get(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "u1" {
		t.Fatalf("user=%s", got.UserID)
	}
}

func TestEventedManager_CreateDelete(t *testing.T) {
	inner, err := sessionmem.New(session.Config{TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var got events.Event
	done := make(chan struct{}, 1)
	_, err = bus.Subscribe(context.Background(), session.TopicSession, func(ctx context.Context, ev events.Event) error {
		got = ev
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	mgr := session.NewEventedManager(inner, bus)
	s, err := mgr.Create(context.Background(), "u2", nil)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for create event")
	}
	if got.Type != session.EventTypeSessionCreated {
		t.Fatalf("type=%s", got.Type)
	}

	if err := mgr.Delete(context.Background(), s.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for delete event")
	}
	if got.Type != session.EventTypeSessionDeleted {
		t.Fatalf("type=%s", got.Type)
	}
}

func TestEventedManager_NilBus(t *testing.T) {
	inner, err := sessionmem.New(session.Config{TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	mgr := session.NewEventedManager(inner, nil)
	if _, err := mgr.Create(context.Background(), "u", nil); err != nil {
		t.Fatal(err)
	}
}
