package memory

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/blob"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type item struct {
	data    []byte
	modTime time.Time
}

// Store implements an in-memory blob store.
type Store struct {
	mu    sync.RWMutex
	items map[string]*item
}

// New creates a new in-memory store.
func New(_ blob.Config) *Store {
	return &Store{
		items: make(map[string]*item),
	}
}

func (s *Store) Upload(ctx context.Context, key string, data io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, data); err != nil {
		return errors.Internal("failed to read data", err)
	}

	s.items[key] = &item{
		data:    buf.Bytes(),
		modTime: time.Now(),
	}
	return nil
}

func (s *Store) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return nil, errors.NotFound("blob not found", nil)
	}

	return io.NopCloser(bytes.NewReader(item.data)), nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return errors.NotFound("blob not found", nil)
	}

	delete(s.items, key)
	return nil
}

func (s *Store) URL(key string) string {
	return "memory://" + key
}
