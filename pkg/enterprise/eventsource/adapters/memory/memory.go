package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/enterprise/eventsource"
)

// Ensure compile-time interface compliance.
var (
	_ eventsource.EventStore    = (*EventStore)(nil)
	_ eventsource.SnapshotStore = (*SnapshotStore)(nil)
)

// EventStore is an in-memory event store for testing.
type EventStore struct {
	streams map[string][]eventsource.Event
	mu      *concurrency.SmartRWMutex
}

// NewEventStore creates a new in-memory event store.
func NewEventStore() *EventStore {
	return &EventStore{
		streams: make(map[string][]eventsource.Event),
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "eventsource-memory"}),
	}
}

// Append adds events to an aggregate's stream with optimistic concurrency.
// Event versions are 1-based (first event is version 1).
func (s *EventStore) Append(ctx context.Context, aggregateID string, expectedVersion int, events []eventsource.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if aggregateID == "" {
		return eventsource.ErrInvalidArgument("aggregateID is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	stream := s.streams[aggregateID]
	currentVersion := len(stream)

	if expectedVersion >= 0 && currentVersion != expectedVersion {
		return eventsource.VersionConflict(aggregateID, expectedVersion, currentVersion)
	}

	for i := range events {
		events[i].AggregateID = aggregateID
		events[i].Version = currentVersion + i + 1
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = time.Now().UTC()
		}
	}

	s.streams[aggregateID] = append(stream, events...)
	return nil
}

// Load retrieves all events for an aggregate.
func (s *EventStore) Load(ctx context.Context, aggregateID string) ([]eventsource.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streams[aggregateID]
	if !ok {
		return []eventsource.Event{}, nil
	}

	result := make([]eventsource.Event, len(stream))
	copy(result, stream)
	return result, nil
}

// LoadFrom retrieves events with Version >= fromVersion (1-based versions).
func (s *EventStore) LoadFrom(ctx context.Context, aggregateID string, fromVersion int) ([]eventsource.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, ok := s.streams[aggregateID]
	if !ok {
		return []eventsource.Event{}, nil
	}

	if fromVersion <= 1 {
		result := make([]eventsource.Event, len(stream))
		copy(result, stream)
		return result, nil
	}

	// Versions are 1-based and sequential: version N is at index N-1.
	startIdx := fromVersion - 1
	if startIdx >= len(stream) {
		return []eventsource.Event{}, nil
	}

	result := make([]eventsource.Event, len(stream)-startIdx)
	copy(result, stream[startIdx:])
	return result, nil
}

// LoadAll retrieves all events across aggregates.
func (s *EventStore) LoadAll(ctx context.Context) ([]eventsource.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []eventsource.Event
	for _, stream := range s.streams {
		all = append(all, stream...)
	}
	return all, nil
}

// SnapshotStore is an in-memory snapshot store.
type SnapshotStore struct {
	snapshots map[string]eventsource.Snapshot
	mu        *concurrency.SmartRWMutex
}

// NewSnapshotStore creates a new in-memory snapshot store.
func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{
		snapshots: make(map[string]eventsource.Snapshot),
		mu:        concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "eventsource-snapshot-memory"}),
	}
}

// Save stores a snapshot.
func (s *SnapshotStore) Save(ctx context.Context, snapshot eventsource.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if snapshot.AggregateID == "" {
		return eventsource.ErrInvalidArgument("aggregateID is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.AggregateID] = snapshot
	return nil
}

// Load retrieves the latest snapshot for an aggregate.
func (s *SnapshotStore) Load(ctx context.Context, aggregateID string) (*eventsource.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot, ok := s.snapshots[aggregateID]
	if !ok {
		return nil, nil
	}
	cp := snapshot
	return &cp, nil
}
