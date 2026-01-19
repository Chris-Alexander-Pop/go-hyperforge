package memory

import (
	"context"
	"sync"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Store implements secrets.Manager in memory.
type Store struct {
	data map[string]string
	mu   sync.RWMutex
}

// New creates a new in-memory secret store.
func New() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) GetSecret(ctx context.Context, key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	val, ok := s.data[key]
	if !ok {
		return "", errors.NotFound("secret not found: "+key, nil)
	}
	return val, nil
}

func (s *Store) SetSecret(ctx context.Context, key string, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *Store) DeleteSecret(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

func (s *Store) Close() error {
	return nil
}
