// Package memory provides an in-memory saga.StateStore for tests and local use.
package memory

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/saga"
)

// Ensure compile-time compliance.
var _ saga.StateStore = (*Store)(nil)

// Store is an in-memory durable saga state store.
type Store struct {
	mu    *concurrency.SmartRWMutex
	items map[string]*saga.PersistedState
}

// New creates an empty memory StateStore.
func New() *Store {
	return &Store{
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "saga-memory-store"}),
		items: make(map[string]*saga.PersistedState),
	}
}

// Save upserts state by ID.
func (s *Store) Save(ctx context.Context, state *saga.PersistedState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if state == nil || state.ID == "" {
		return errors.InvalidArgument("saga state id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[state.ID] = state.Clone()
	return nil
}

// Load returns a clone of stored state.
func (s *Store) Load(ctx context.Context, executionID string) (*saga.PersistedState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.items[executionID]
	if !ok {
		return nil, errors.NotFound("saga execution not found", nil)
	}
	return st.Clone(), nil
}

// Delete removes state by ID.
func (s *Store) Delete(ctx context.Context, executionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[executionID]; !ok {
		return errors.NotFound("saga execution not found", nil)
	}
	delete(s.items, executionID)
	return nil
}

// ListIncomplete returns non-terminal executions.
func (s *Store) ListIncomplete(ctx context.Context) ([]*saga.PersistedState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*saga.PersistedState, 0)
	for _, st := range s.items {
		if !st.IsTerminal() {
			out = append(out, st.Clone())
		}
	}
	return out, nil
}
