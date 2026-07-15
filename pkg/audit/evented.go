package audit

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedStore implements Store at compile time.
var _ Store = (*EventedStore)(nil)

const (
	// TopicAudit is the pkg/events topic for audit fan-out (reactive subscribers).
	// Note: audit stores are themselves an event log; this decorator optionally
	// mirrors Append into pkg/events for cross-domain reaction.
	TopicAudit = "audit"

	// EventTypeRecorded is emitted after a successful Append.
	EventTypeRecorded = "audit.recorded"
)

// RecordedPayload is a compact payload for audit.recorded events.
type RecordedPayload struct {
	EventType EventType `json:"event_type"`
	ActorID   string    `json:"actor_id,omitempty"`
	Outcome   Outcome   `json:"outcome,omitempty"`
	Action    string    `json:"action,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EventedStore decorates a Store to publish domain events via pkg/events after Append.
// Publish is best-effort: failures are ignored so audit writes are not rolled back.
type EventedStore struct {
	next Store
	bus  events.Bus
}

// NewEventedStore wraps next so Append fans out to bus after success.
// If bus is nil, publishing is skipped.
func NewEventedStore(next Store, bus events.Bus) *EventedStore {
	return &EventedStore{next: next, bus: bus}
}

func (s *EventedStore) publish(ctx context.Context, event Event) {
	if s.bus == nil {
		return
	}
	id := event.ID
	if id == "" {
		id = uuid.NewString()
	}
	_ = s.bus.Publish(ctx, TopicAudit, events.Event{
		ID:        id,
		Type:      EventTypeRecorded,
		Source:    "pkg/audit",
		Timestamp: time.Now().UTC(),
		Payload: RecordedPayload{
			EventType: event.EventType,
			ActorID:   event.ActorID,
			Outcome:   event.Outcome,
			Action:    event.Action,
			Timestamp: event.Timestamp,
		},
	})
}

// Append delegates then publishes audit.recorded (best-effort).
func (s *EventedStore) Append(ctx context.Context, event Event) error {
	if err := s.next.Append(ctx, event); err != nil {
		return err
	}
	s.publish(ctx, event)
	return nil
}

// Query delegates to the underlying store.
func (s *EventedStore) Query(ctx context.Context, filter QueryFilter) ([]Event, error) {
	return s.next.Query(ctx, filter)
}
