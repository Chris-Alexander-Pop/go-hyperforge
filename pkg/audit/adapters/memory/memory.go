/*
Package memory provides an in-memory audit.Store for tests and local development.

Uses pkg/concurrency.SmartRWMutex for observability-friendly locking.
Events are retained only for the process lifetime.
*/
package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Ensure compile-time interface compliance.
var _ audit.Store = (*Store)(nil)

// Store is an in-memory audit event store.
type Store struct {
	events []audit.Event
	mu     *concurrency.SmartRWMutex
}

// NewStore creates an empty in-memory audit store.
func NewStore() *Store {
	return &Store{
		events: make([]audit.Event, 0),
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "audit-memory"}),
	}
}

// Append records an audit event.
func (s *Store) Append(ctx context.Context, event audit.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.EventType == "" {
		return audit.ErrInvalidArgument("event_type is required", nil)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Copy metadata map so callers cannot mutate stored events.
	if event.Metadata != nil {
		cp := make(map[string]interface{}, len(event.Metadata))
		for k, v := range event.Metadata {
			cp[k] = v
		}
		event.Metadata = cp
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

// Query returns events matching the filter (actor, type, time range, limit).
func (s *Store) Query(ctx context.Context, filter audit.QueryFilter) ([]audit.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if filter.Limit < 0 {
		return nil, audit.ErrInvalidArgument("limit must be >= 0", nil)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]audit.Event, 0)
	for _, e := range s.events {
		if filter.ActorID != "" && e.ActorID != filter.ActorID {
			continue
		}
		if filter.EventType != "" && e.EventType != filter.EventType {
			continue
		}
		if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && e.Timestamp.After(filter.Until) {
			continue
		}
		out = append(out, e)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

// Len returns the number of stored events (test helper).
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}
