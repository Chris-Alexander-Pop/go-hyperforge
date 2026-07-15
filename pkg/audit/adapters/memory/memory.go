/*
Package memory provides an in-memory audit.Store for tests and local development.

Uses pkg/concurrency.SmartRWMutex for observability-friendly locking.
Events are retained only for the process lifetime.

Optional hash chaining (NewChainedStore / WithHashChain) stamps ID, Hash, and
PrevHash for tamper-evident append-only logs. Lifecycle methods support
retention purge and GDPR Export/Erase by actor ID.
*/
package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/google/uuid"
)

// Ensure compile-time interface compliance.
var (
	_ audit.Store          = (*Store)(nil)
	_ audit.LifecycleStore = (*Store)(nil)
)

// Store is an in-memory audit event store.
type Store struct {
	events    []audit.Event
	mu        *concurrency.SmartRWMutex
	hashChain bool
	lastHash  string
}

// Option configures a memory Store.
type Option func(*Store)

// WithHashChain enables tamper-evident hash chaining on Append.
func WithHashChain(enabled bool) Option {
	return func(s *Store) {
		s.hashChain = enabled
		if enabled && s.lastHash == "" {
			s.lastHash = "GENESIS"
		}
	}
}

// NewStore creates an empty in-memory audit store.
func NewStore(opts ...Option) *Store {
	s := &Store{
		events: make([]audit.Event, 0),
		mu:     concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "audit-memory"}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewChainedStore creates a memory store with hash chaining enabled.
func NewChainedStore() *Store {
	return NewStore(WithHashChain(true))
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

	if s.hashChain {
		if event.ID == "" {
			event.ID = uuid.NewString()
		}
		event.PrevHash = s.lastHash
		h, err := audit.HashEvent(event)
		if err != nil {
			return err
		}
		event.Hash = h
		s.lastHash = h
	}

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
		if !matchFilter(e, filter) {
			continue
		}
		out = append(out, e)
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func matchFilter(e audit.Event, filter audit.QueryFilter) bool {
	if filter.ActorID != "" && e.ActorID != filter.ActorID {
		return false
	}
	if filter.EventType != "" && e.EventType != filter.EventType {
		return false
	}
	if !filter.Since.IsZero() && e.Timestamp.Before(filter.Since) {
		return false
	}
	if !filter.Until.IsZero() && e.Timestamp.After(filter.Until) {
		return false
	}
	return true
}

// Purge deletes events older than olderThan.
func (s *Store) Purge(ctx context.Context, olderThan time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if olderThan.IsZero() {
		return 0, audit.ErrInvalidArgument("olderThan is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.events[:0]
	var removed int64
	for _, e := range s.events {
		if e.Timestamp.Before(olderThan) {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	s.events = kept
	return removed, nil
}

// ExportByActor returns all events for actorID.
func (s *Store) ExportByActor(ctx context.Context, actorID string) ([]audit.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if actorID == "" {
		return nil, audit.ErrInvalidArgument("actorID is required", nil)
	}
	return s.Query(ctx, audit.QueryFilter{ActorID: actorID})
}

// EraseByActor permanently deletes events for actorID.
func (s *Store) EraseByActor(ctx context.Context, actorID string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if actorID == "" {
		return 0, audit.ErrInvalidArgument("actorID is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	kept := s.events[:0]
	var removed int64
	for _, e := range s.events {
		if e.ActorID == actorID {
			removed++
			continue
		}
		kept = append(kept, e)
	}
	s.events = kept
	return removed, nil
}

// Len returns the number of stored events (test helper).
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

// LastHash returns the current chain tip (empty when chaining is disabled).
func (s *Store) LastHash() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hashChain {
		return ""
	}
	return s.lastHash
}
