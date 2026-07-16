// Package memstore provides a small in-memory record store for Hyperforge services.
package memstore

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Record is a generic persisted entity.
type Record struct {
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Store is an in-memory record store.
type Store struct {
	mu   sync.RWMutex
	byID map[string]Record
}

// New creates an empty store.
func New() *Store {
	return &Store{byID: make(map[string]Record)}
}

// Create inserts a new record (generates ID when empty).
func (s *Store) Create(ctx context.Context, data map[string]interface{}) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	now := time.Now().UTC()
	rec := Record{
		ID:        uuid.NewString(),
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if id, ok := data["id"].(string); ok && id != "" {
		rec.ID = id
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.byID[rec.ID]; exists {
		return Record{}, errors.Conflict("record already exists", nil)
	}
	s.byID[rec.ID] = rec
	return rec, nil
}

// Get returns a record by ID.
func (s *Store) Get(ctx context.Context, id string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if id == "" {
		return Record{}, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.byID[id]
	if !ok {
		return Record{}, errors.NotFound("record not found", nil)
	}
	return rec, nil
}

// List returns all records (copy).
func (s *Store) List(ctx context.Context) ([]Record, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(s.byID))
	for _, rec := range s.byID {
		out = append(out, rec)
	}
	return out, nil
}

// Update merges data into an existing record.
func (s *Store) Update(ctx context.Context, id string, data map[string]interface{}) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.byID[id]
	if !ok {
		return Record{}, errors.NotFound("record not found", nil)
	}
	if rec.Data == nil {
		rec.Data = map[string]interface{}{}
	}
	for k, v := range data {
		if k == "id" {
			continue
		}
		rec.Data[k] = v
	}
	rec.UpdatedAt = time.Now().UTC()
	s.byID[id] = rec
	return rec, nil
}

// Delete removes a record.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return errors.NotFound("record not found", nil)
	}
	delete(s.byID, id)
	return nil
}
