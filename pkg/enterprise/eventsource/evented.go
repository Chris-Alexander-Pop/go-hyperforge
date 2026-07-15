package eventsource

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedStore implements EventStore at compile time.
var _ EventStore = (*EventedStore)(nil)

// EventedStore decorates an EventStore to publish committed events onto a
// pkg/events.Bus after a successful Append.
//
// Topic is the aggregate type when set, otherwise "aggregates".
// Event type is the stream EventType. Payload is the stored Event.
type EventedStore struct {
	next EventStore
	bus  events.Bus
}

// NewEventedStore wraps next so Append fans out to bus after persistence.
// If bus is nil, NewEventedStore returns next unchanged behavior via a no-op
// wrapper that still delegates; prefer passing a real bus.
func NewEventedStore(next EventStore, bus events.Bus) *EventedStore {
	return &EventedStore{next: next, bus: bus}
}

// Append persists events then publishes each to pkg/events (best-effort).
func (s *EventedStore) Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error {
	if err := s.next.Append(ctx, aggregateID, expectedVersion, events); err != nil {
		return err
	}
	if s.bus == nil {
		return nil
	}
	for i := range events {
		_ = s.publish(ctx, events[i])
	}
	return nil
}

func (s *EventedStore) publish(ctx context.Context, e Event) error {
	topic := e.AggregateType
	if topic == "" {
		topic = "aggregates"
	}
	id := e.ID
	if id == "" {
		id = uuid.NewString()
	}
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return s.bus.Publish(ctx, topic, events.Event{
		ID:        id,
		Type:      e.EventType,
		Source:    "pkg/enterprise/eventsource",
		Timestamp: ts,
		Payload:   e,
	})
}

// Load delegates to the underlying store.
func (s *EventedStore) Load(ctx context.Context, aggregateID string) ([]Event, error) {
	return s.next.Load(ctx, aggregateID)
}

// LoadFrom delegates to the underlying store.
func (s *EventedStore) LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error) {
	return s.next.LoadFrom(ctx, aggregateID, fromVersion)
}

// LoadAll delegates to the underlying store.
func (s *EventedStore) LoadAll(ctx context.Context) ([]Event, error) {
	return s.next.LoadAll(ctx)
}
