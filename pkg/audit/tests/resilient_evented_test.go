package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/audit"
	auditmem "github.com/chris-alexander-pop/go-hyperforge/pkg/audit/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmem "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestResilientStore_AppendQuery(t *testing.T) {
	store := audit.NewResilientStore(auditmem.NewStore(), audit.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	ctx := context.Background()
	ev := audit.Event{
		EventType: audit.EventTypeLogin,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "u1",
		Timestamp: time.Now().UTC(),
	}
	if err := store.Append(ctx, ev); err != nil {
		t.Fatal(err)
	}
	got, err := store.Query(ctx, audit.QueryFilter{ActorID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestEventedStore_Append(t *testing.T) {
	bus := eventsmem.New(events.Config{})
	t.Cleanup(func() { _ = bus.Close() })

	var got events.Event
	done := make(chan struct{}, 1)
	_, err := bus.Subscribe(context.Background(), audit.TopicAudit, func(ctx context.Context, ev events.Event) error {
		got = ev
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	store := audit.NewEventedStore(auditmem.NewStore(), bus)
	if err := store.Append(context.Background(), audit.Event{
		EventType: audit.EventTypeDataCreate,
		Outcome:   audit.OutcomeSuccess,
		ActorID:   "a1",
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	if got.Type != audit.EventTypeRecorded {
		t.Fatalf("type=%s", got.Type)
	}
	payload, ok := got.Payload.(audit.RecordedPayload)
	if !ok {
		t.Fatalf("payload %T", got.Payload)
	}
	if payload.ActorID != "a1" {
		t.Fatalf("actor=%s", payload.ActorID)
	}
}

func TestEventedStore_NilBus(t *testing.T) {
	store := audit.NewEventedStore(auditmem.NewStore(), nil)
	if err := store.Append(context.Background(), audit.Event{
		EventType: audit.EventTypeLogin,
		Outcome:   audit.OutcomeSuccess,
		Timestamp: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
}
