package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
	"github.com/google/uuid"
)

// Identity is a local identity record with roles.
type Identity struct {
	ID           string
	Username     string
	Email        string
	Roles        []string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateInput creates an identity.
type CreateInput struct {
	Username string
	Email    string
	Roles    []string
	Password string
}

// UpdateInput updates mutable identity fields.
type UpdateInput struct {
	Email    string
	Roles    []string
	Password string
}

// Store is an in-memory identity store.
type Store struct {
	mu         sync.RWMutex
	identities map[string]*Identity
	byUser     map[string]string
	hasher     *crypto.Hasher
}

// New creates an empty identity store.
func New() *Store {
	return &Store{
		identities: make(map[string]*Identity),
		byUser:     make(map[string]string),
		hasher:     crypto.NewHasher(crypto.DefaultHashConfig()),
	}
}

// Create inserts a new identity.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Identity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Username == "" {
		return nil, errors.InvalidArgument("username is required", nil)
	}

	var hash string
	if in.Password != "" {
		h, err := s.hasher.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		hash = h
	}

	now := time.Now().UTC()
	id := &Identity{
		ID:           uuid.NewString(),
		Username:     in.Username,
		Email:        in.Email,
		Roles:        append([]string(nil), in.Roles...),
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byUser[in.Username]; exists {
		return nil, errors.Conflict("username already exists", nil)
	}
	s.identities[id.ID] = id
	s.byUser[in.Username] = id.ID
	return clone(id), nil
}

// Get returns an identity by ID.
func (s *Store) Get(ctx context.Context, id string) (*Identity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ident, ok := s.identities[id]
	if !ok {
		return nil, errors.NotFound("identity not found", nil)
	}
	return clone(ident), nil
}

// List returns all identities.
func (s *Store) List(ctx context.Context) ([]*Identity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Identity, 0, len(s.identities))
	for _, id := range s.identities {
		out = append(out, clone(id))
	}
	return out, nil
}

// Update updates an identity.
func (s *Store) Update(ctx context.Context, id string, in UpdateInput) (*Identity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	ident, ok := s.identities[id]
	if !ok {
		return nil, errors.NotFound("identity not found", nil)
	}
	if in.Email != "" {
		ident.Email = in.Email
	}
	if in.Roles != nil {
		ident.Roles = append([]string(nil), in.Roles...)
	}
	if in.Password != "" {
		h, err := s.hasher.Hash(in.Password)
		if err != nil {
			return nil, err
		}
		ident.PasswordHash = h
	}
	ident.UpdatedAt = time.Now().UTC()
	return clone(ident), nil
}

// Delete removes an identity.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ident, ok := s.identities[id]
	if !ok {
		return errors.NotFound("identity not found", nil)
	}
	delete(s.byUser, ident.Username)
	delete(s.identities, id)
	return nil
}

func clone(id *Identity) *Identity {
	cp := *id
	cp.Roles = append([]string(nil), id.Roles...)
	return &cp
}
