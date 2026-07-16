package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// RequestType is the GDPR request kind.
type RequestType string

const (
	TypeErasure RequestType = "erasure"
	TypeExport  RequestType = "export"
)

// Status is the GDPR request lifecycle state.
type Status string

const (
	StatusPending    Status = "pending"
	StatusCompleted  Status = "completed"
	StatusInProgress Status = "in_progress"
)

// Request is a GDPR data subject request.
type Request struct {
	ID        string
	Type      RequestType
	SubjectID string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateInput creates a GDPR request.
type CreateInput struct {
	Type      RequestType
	SubjectID string
}

// Store is an in-memory GDPR request store.
type Store struct {
	mu   sync.RWMutex
	reqs map[string]*Request
}

// New creates an empty GDPR store.
func New() *Store {
	return &Store{reqs: make(map[string]*Request)}
}

// Create stores a new pending request.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.SubjectID == "" {
		return nil, errors.InvalidArgument("subject_id is required", nil)
	}
	if in.Type != TypeErasure && in.Type != TypeExport {
		return nil, errors.InvalidArgument("type must be erasure or export", nil)
	}

	now := time.Now().UTC()
	req := &Request{
		ID:        uuid.NewString(),
		Type:      in.Type,
		SubjectID: in.SubjectID,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs[req.ID] = req
	return clone(req), nil
}

// Get returns a request by ID.
func (s *Store) Get(ctx context.Context, id string) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.reqs[id]
	if !ok {
		return nil, errors.NotFound("gdpr request not found", nil)
	}
	return clone(req), nil
}

// Complete marks a request as completed.
func (s *Store) Complete(ctx context.Context, id string) (*Request, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.reqs[id]
	if !ok {
		return nil, errors.NotFound("gdpr request not found", nil)
	}
	if req.Status == StatusCompleted {
		return nil, errors.FailedPrecondition("request already completed", nil)
	}
	req.Status = StatusCompleted
	req.UpdatedAt = time.Now().UTC()
	return clone(req), nil
}

func clone(r *Request) *Request {
	cp := *r
	return &cp
}
