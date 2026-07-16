package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Check is a recorded compliance check.
type Check struct {
	ID        string
	Policy    string
	SubjectID string
	Result    string
	CreatedAt time.Time
}

// RecordInput records a compliance check.
type RecordInput struct {
	Policy    string
	SubjectID string
	Result    string
}

// Store is an in-memory compliance check store.
type Store struct {
	mu     sync.RWMutex
	checks map[string]*Check
	order  []string
}

// New creates an empty compliance store.
func New() *Store {
	return &Store{checks: make(map[string]*Check)}
}

// Record stores a compliance check.
func (s *Store) Record(ctx context.Context, in RecordInput) (*Check, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Policy == "" {
		return nil, errors.InvalidArgument("policy is required", nil)
	}
	if in.SubjectID == "" {
		return nil, errors.InvalidArgument("subject_id is required", nil)
	}
	if in.Result == "" {
		return nil, errors.InvalidArgument("result is required", nil)
	}

	ch := &Check{
		ID:        uuid.NewString(),
		Policy:    in.Policy,
		SubjectID: in.SubjectID,
		Result:    in.Result,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks[ch.ID] = ch
	s.order = append(s.order, ch.ID)
	cp := *ch
	return &cp, nil
}

// Get returns a check by ID.
func (s *Store) Get(ctx context.Context, id string) (*Check, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.checks[id]
	if !ok {
		return nil, errors.NotFound("compliance check not found", nil)
	}
	cp := *ch
	return &cp, nil
}

// List returns all checks in insertion order.
func (s *Store) List(ctx context.Context) ([]*Check, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Check, 0, len(s.order))
	for _, id := range s.order {
		ch := s.checks[id]
		cp := *ch
		out = append(out, &cp)
	}
	return out, nil
}
