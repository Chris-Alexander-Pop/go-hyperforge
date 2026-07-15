// Package eventsource provides Event Sourcing patterns.
//
// Stores state as a sequence of events rather than current state snapshots.
//
// Usage:
//
//	import (
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
//	    "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource/adapters/memory"
//	)
//
//	store := memory.NewEventStore()
//	err := store.Append(ctx, "order-123", 0, events)
//	history, err := store.Load(ctx, "order-123")
package eventsource

import (
	"context"
	"encoding/json"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Event represents a stored event.
// Versions are 1-based: the first event in a stream has Version 1.
type Event struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	EventType     string                 `json:"event_type"`
	Version       int                    `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	Data          json.RawMessage        `json:"data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EventStore persists and retrieves events.
type EventStore interface {
	// Append adds events to an aggregate's stream.
	// expectedVersion is the current stream version (0 for a new stream).
	// Pass expectedVersion < 0 to skip the optimistic concurrency check.
	Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error

	// Load retrieves all events for an aggregate.
	Load(ctx context.Context, aggregateID string) ([]Event, error)

	// LoadFrom retrieves events with Version >= fromVersion (1-based).
	// fromVersion <= 1 returns the full stream (same as Load for existing streams).
	LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)

	// LoadAll retrieves all events (for projections).
	LoadAll(ctx context.Context) ([]Event, error)
}

// EventSourcedAggregate can be reconstructed from events.
type EventSourcedAggregate interface {
	// AggregateID returns the aggregate identifier.
	AggregateID() string

	// AggregateType returns the aggregate type name.
	AggregateType() string

	// Version returns the current committed version (0 before any events).
	Version() int

	// SetVersion sets the committed version after load or save.
	SetVersion(version int)

	// ApplyEvent applies an event to update state.
	ApplyEvent(event Event) error

	// GetUncommittedEvents returns events not yet persisted.
	GetUncommittedEvents() []Event

	// ClearUncommittedEvents clears uncommitted events.
	ClearUncommittedEvents()
}

// BaseEventSourcedAggregate provides common functionality.
type BaseEventSourcedAggregate struct {
	id                string
	aggregateType     string
	version           int
	uncommittedEvents []Event
}

// NewBaseEventSourcedAggregate creates a new base aggregate.
func NewBaseEventSourcedAggregate(id, aggregateType string) BaseEventSourcedAggregate {
	return BaseEventSourcedAggregate{
		id:                id,
		aggregateType:     aggregateType,
		version:           0,
		uncommittedEvents: make([]Event, 0),
	}
}

// AggregateID returns the aggregate identifier.
func (a *BaseEventSourcedAggregate) AggregateID() string {
	return a.id
}

// AggregateType returns the aggregate type.
func (a *BaseEventSourcedAggregate) AggregateType() string {
	return a.aggregateType
}

// Version returns the current version.
func (a *BaseEventSourcedAggregate) Version() int {
	return a.version
}

// SetVersion sets the committed version.
func (a *BaseEventSourcedAggregate) SetVersion(version int) {
	a.version = version
}

// IncrementVersion increments the version.
func (a *BaseEventSourcedAggregate) IncrementVersion() {
	a.version++
}

// GetUncommittedEvents returns uncommitted events.
func (a *BaseEventSourcedAggregate) GetUncommittedEvents() []Event {
	return a.uncommittedEvents
}

// ClearUncommittedEvents clears uncommitted events.
func (a *BaseEventSourcedAggregate) ClearUncommittedEvents() {
	a.uncommittedEvents = make([]Event, 0)
}

// RecordEvent records an event for later persistence.
func (a *BaseEventSourcedAggregate) RecordEvent(eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return pkgerrors.Internal("failed to marshal event data", err)
	}

	event := Event{
		AggregateID:   a.id,
		AggregateType: a.aggregateType,
		EventType:     eventType,
		Version:       a.version + len(a.uncommittedEvents) + 1,
		Timestamp:     time.Now(),
		Data:          jsonData,
	}

	a.uncommittedEvents = append(a.uncommittedEvents, event)
	return nil
}

// EventRepository provides aggregate persistence through events.
type EventRepository struct {
	store     EventStore
	snapshots SnapshotStore
}

// NewEventRepository creates a new event repository without snapshot support.
func NewEventRepository(store EventStore) *EventRepository {
	return &EventRepository{store: store}
}

// NewEventRepositoryWithSnapshots creates a repository that loads via snapshots
// when the aggregate implements Snapshottable.
func NewEventRepositoryWithSnapshots(store EventStore, snapshots SnapshotStore) *EventRepository {
	return &EventRepository{store: store, snapshots: snapshots}
}

// SetSnapshotStore attaches or replaces the optional SnapshotStore.
func (r *EventRepository) SetSnapshotStore(snapshots SnapshotStore) {
	r.snapshots = snapshots
}

// Save persists uncommitted events and advances the aggregate version.
func (r *EventRepository) Save(ctx context.Context, aggregate EventSourcedAggregate) error {
	events := aggregate.GetUncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	err := r.store.Append(ctx, aggregate.AggregateID(), aggregate.Version(), events)
	if err != nil {
		return err
	}

	// Append assigns 1-based versions onto the event slice in place.
	aggregate.SetVersion(events[len(events)-1].Version)
	aggregate.ClearUncommittedEvents()
	return nil
}

// Load reconstructs an aggregate from its event history and sets its version.
// When a SnapshotStore is configured and aggregate implements Snapshottable,
// Load restores from the latest snapshot then applies only later events.
func (r *EventRepository) Load(ctx context.Context, aggregate EventSourcedAggregate) error {
	if snapAgg, ok := aggregate.(Snapshottable); ok && r.snapshots != nil {
		snap, err := r.snapshots.Load(ctx, aggregate.AggregateID())
		if err != nil {
			return err
		}
		if snap != nil {
			if err := snapAgg.RestoreSnapshot(snap.Data); err != nil {
				return err
			}
			aggregate.SetVersion(snap.Version)
			return r.applyFrom(ctx, aggregate, snap.Version+1)
		}
	}

	events, err := r.store.Load(ctx, aggregate.AggregateID())
	if err != nil {
		return err
	}
	return r.applyEvents(aggregate, events)
}

func (r *EventRepository) applyFrom(ctx context.Context, aggregate EventSourcedAggregate, fromVersion int) error {
	events, err := r.store.LoadFrom(ctx, aggregate.AggregateID(), fromVersion)
	if err != nil {
		return err
	}
	return r.applyEvents(aggregate, events)
}

func (r *EventRepository) applyEvents(aggregate EventSourcedAggregate, events []Event) error {
	for _, event := range events {
		if err := aggregate.ApplyEvent(event); err != nil {
			return err
		}
		aggregate.SetVersion(event.Version)
	}
	return nil
}

// SaveSnapshot persists the current aggregate state at Version().
func (r *EventRepository) SaveSnapshot(ctx context.Context, aggregate Snapshottable) error {
	if r.snapshots == nil {
		return ErrInvalidArgument("snapshot store is required", nil)
	}
	data, err := aggregate.SnapshotData()
	if err != nil {
		return err
	}
	return r.snapshots.Save(ctx, Snapshot{
		AggregateID:   aggregate.AggregateID(),
		AggregateType: aggregate.AggregateType(),
		Version:       aggregate.Version(),
		Timestamp:     time.Now().UTC(),
		Data:          data,
	})
}

// Snapshot represents a point-in-time aggregate state.
type Snapshot struct {
	AggregateID   string          `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Version       int             `json:"version"`
	Timestamp     time.Time       `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
}

// SnapshotStore persists and retrieves snapshots.
type SnapshotStore interface {
	// Save stores a snapshot.
	Save(ctx context.Context, snapshot Snapshot) error

	// Load retrieves the latest snapshot for an aggregate.
	// Returns nil, nil when no snapshot exists.
	Load(ctx context.Context, aggregateID string) (*Snapshot, error)
}

// Snapshottable is an EventSourcedAggregate that can serialize and restore
// its state for SnapshotStore-backed loads.
type Snapshottable interface {
	EventSourcedAggregate
	RestoreSnapshot(data json.RawMessage) error
	SnapshotData() (json.RawMessage, error)
}
