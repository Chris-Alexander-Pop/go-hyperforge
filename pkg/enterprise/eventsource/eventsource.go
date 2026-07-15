package eventsource

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Event represents a stored event.
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
	Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error

	// Load retrieves all events for an aggregate.
	Load(ctx context.Context, aggregateID string) ([]Event, error)

	// LoadFrom retrieves events from a specific version.
	LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)

	// LoadAll retrieves all events (for projections).
	LoadAll(ctx context.Context) ([]Event, error)
}

// InMemoryEventStore is an in-memory event store for testing.
type InMemoryEventStore struct {
	streams map[string][]Event
	mu      sync.RWMutex
}

// NewInMemoryEventStore creates a new in-memory event store.
func NewInMemoryEventStore() *InMemoryEventStore {
	return &InMemoryEventStore{
		streams: make(map[string][]Event),
	}
}

// Append adds events to an aggregate's stream.
func (s *InMemoryEventStore) Append(ctx context.Context, aggregateID string, expectedVersion int, events []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream := s.streams[aggregateID]
	currentVersion := len(stream)

	// Optimistic concurrency check
	if expectedVersion >= 0 && currentVersion != expectedVersion {
		return pkgerrors.Conflict("version conflict", nil)
	}

	// Set versions and timestamps
	for i := range events {
		events[i].Version = currentVersion + i + 1
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = time.Now()
		}
	}

	s.streams[aggregateID] = append(stream, events...)
	return nil
}

// Load retrieves all events for an aggregate.
func (s *InMemoryEventStore) Load(ctx context.Context, aggregateID string) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streams[aggregateID]
	if !ok {
		return []Event{}, nil
	}

	// Return a copy
	result := make([]Event, len(stream))
	copy(result, stream)
	return result, nil
}

// LoadFrom retrieves events from a specific version.
func (s *InMemoryEventStore) LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streams[aggregateID]
	if !ok {
		return []Event{}, nil
	}

	if fromVersion >= len(stream) {
		return []Event{}, nil
	}

	result := make([]Event, len(stream)-fromVersion)
	copy(result, stream[fromVersion:])
	return result, nil
}

// LoadAll retrieves all events.
func (s *InMemoryEventStore) LoadAll(ctx context.Context) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []Event
	for _, stream := range s.streams {
		all = append(all, stream...)
	}

	return all, nil
}

// EventSourcedAggregate can be reconstructed from events.
type EventSourcedAggregate interface {
	// AggregateID returns the aggregate identifier.
	AggregateID() string

	// AggregateType returns the aggregate type name.
	AggregateType() string

	// Version returns the current version.
	Version() int

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
	store EventStore
}

// NewEventRepository creates a new event repository.
func NewEventRepository(store EventStore) *EventRepository {
	return &EventRepository{store: store}
}

// Save persists uncommitted events.
func (r *EventRepository) Save(ctx context.Context, aggregate EventSourcedAggregate) error {
	events := aggregate.GetUncommittedEvents()
	if len(events) == 0 {
		return nil
	}

	err := r.store.Append(ctx, aggregate.AggregateID(), aggregate.Version(), events)
	if err != nil {
		return err
	}

	aggregate.ClearUncommittedEvents()
	return nil
}

// Load reconstructs an aggregate from its event history.
func (r *EventRepository) Load(ctx context.Context, aggregate EventSourcedAggregate) error {
	events, err := r.store.Load(ctx, aggregate.AggregateID())
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := aggregate.ApplyEvent(event); err != nil {
			return err
		}
	}

	return nil
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
	Load(ctx context.Context, aggregateID string) (*Snapshot, error)
}

// InMemorySnapshotStore is an in-memory snapshot store.
type InMemorySnapshotStore struct {
	snapshots map[string]Snapshot
	mu        sync.RWMutex
}

// NewInMemorySnapshotStore creates a new in-memory snapshot store.
func NewInMemorySnapshotStore() *InMemorySnapshotStore {
	return &InMemorySnapshotStore{
		snapshots: make(map[string]Snapshot),
	}
}

// Save stores a snapshot.
func (s *InMemorySnapshotStore) Save(ctx context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.AggregateID] = snapshot
	return nil
}

// Load retrieves the latest snapshot.
func (s *InMemorySnapshotStore) Load(ctx context.Context, aggregateID string) (*Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[aggregateID]
	if !ok {
		return nil, nil
	}
	return &snapshot, nil
}
