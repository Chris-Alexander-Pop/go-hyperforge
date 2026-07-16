package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Profile is a user profile record.
type Profile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is an in-memory profile store.
type Store struct {
	mu   sync.RWMutex
	byID map[string]Profile
}

// New creates an empty profile store.
func New() *Store {
	return &Store{byID: make(map[string]Profile)}
}

// Upsert creates or updates a profile by ID.
func (s *Store) Upsert(ctx context.Context, p Profile) (Profile, error) {
	if err := ctx.Err(); err != nil {
		return Profile{}, err
	}
	if p.ID == "" || p.Email == "" {
		return Profile{}, errors.InvalidArgument("id and email are required", nil)
	}
	if p.Name == "" {
		p.Name = p.Email
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.byID[p.ID]; ok {
		p.CreatedAt = existing.CreatedAt
	} else if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	s.byID[p.ID] = p
	return p, nil
}

// Get returns a profile by ID.
func (s *Store) Get(ctx context.Context, id string) (Profile, error) {
	if err := ctx.Err(); err != nil {
		return Profile{}, err
	}
	if id == "" {
		return Profile{}, errors.InvalidArgument("id is required", nil)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	if !ok {
		return Profile{}, errors.NotFound("user not found", nil)
	}
	return p, nil
}
