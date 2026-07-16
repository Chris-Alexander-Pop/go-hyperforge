package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Status is the KYC application lifecycle state.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

// Application is a KYC verification application.
type Application struct {
	ID        string
	SubjectID string
	FullName  string
	Document  string
	Status    Status
	Reason    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SubmitInput creates a KYC application.
type SubmitInput struct {
	SubjectID string
	FullName  string
	Document  string
}

// Store is an in-memory KYC application store.
type Store struct {
	mu   sync.RWMutex
	apps map[string]*Application
}

// New creates an empty KYC store.
func New() *Store {
	return &Store{apps: make(map[string]*Application)}
}

// Submit stores a new pending application.
func (s *Store) Submit(ctx context.Context, in SubmitInput) (*Application, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.SubjectID == "" {
		return nil, errors.InvalidArgument("subject_id is required", nil)
	}
	if in.FullName == "" {
		return nil, errors.InvalidArgument("full_name is required", nil)
	}

	now := time.Now().UTC()
	app := &Application{
		ID:        uuid.NewString(),
		SubjectID: in.SubjectID,
		FullName:  in.FullName,
		Document:  in.Document,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.apps[app.ID] = app
	return clone(app), nil
}

// Get returns an application by ID.
func (s *Store) Get(ctx context.Context, id string) (*Application, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, errors.NotFound("kyc application not found", nil)
	}
	return clone(app), nil
}

// Approve marks a pending application as approved.
func (s *Store) Approve(ctx context.Context, id string) (*Application, error) {
	return s.decide(ctx, id, StatusApproved, "")
}

// Reject marks a pending application as rejected.
func (s *Store) Reject(ctx context.Context, id, reason string) (*Application, error) {
	return s.decide(ctx, id, StatusRejected, reason)
}

func (s *Store) decide(ctx context.Context, id string, status Status, reason string) (*Application, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	app, ok := s.apps[id]
	if !ok {
		return nil, errors.NotFound("kyc application not found", nil)
	}
	if app.Status != StatusPending {
		return nil, errors.FailedPrecondition("application is not pending", nil)
	}
	app.Status = status
	app.Reason = reason
	app.UpdatedAt = time.Now().UTC()
	return clone(app), nil
}

func clone(a *Application) *Application {
	cp := *a
	return &cp
}
