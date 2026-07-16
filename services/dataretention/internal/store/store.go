package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Policy is a data retention policy.
type Policy struct {
	ID        string
	Resource  string
	Days      int
	CreatedAt time.Time
}

// CreateInput creates a retention policy.
type CreateInput struct {
	Resource string
	Days     int
}

// EvaluateResult is the stub outcome of evaluating expired data.
type EvaluateResult struct {
	PoliciesEvaluated int
	ExpiredDeleted    int
	EvaluatedAt       time.Time
}

// Store is an in-memory retention policy store.
type Store struct {
	mu       sync.RWMutex
	policies map[string]*Policy
	order    []string
}

// New creates an empty retention store.
func New() *Store {
	return &Store{policies: make(map[string]*Policy)}
}

// Create stores a retention policy.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Resource == "" {
		return nil, errors.InvalidArgument("resource is required", nil)
	}
	if in.Days <= 0 {
		return nil, errors.InvalidArgument("days must be positive", nil)
	}

	p := &Policy{
		ID:        uuid.NewString(),
		Resource:  in.Resource,
		Days:      in.Days,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[p.ID] = p
	s.order = append(s.order, p.ID)
	cp := *p
	return &cp, nil
}

// List returns all policies in insertion order.
func (s *Store) List(ctx context.Context) ([]*Policy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Policy, 0, len(s.order))
	for _, id := range s.order {
		p := s.policies[id]
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

// Evaluate is a stub that reports how many policies were considered.
// No real data purge occurs; ExpiredDeleted is always 0.
func (s *Store) Evaluate(ctx context.Context) (*EvaluateResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &EvaluateResult{
		PoliciesEvaluated: len(s.policies),
		ExpiredDeleted:    0,
		EvaluatedAt:       time.Now().UTC(),
	}, nil
}
