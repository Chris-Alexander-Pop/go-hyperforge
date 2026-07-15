package memory

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/storage/blob"
)

// Ensure Store implements blob.Store.
var _ blob.Store = (*Store)(nil)

type item struct {
	data    []byte
	modTime time.Time
}

// Store implements an in-memory blob store.
type Store struct {
	mu    *concurrency.SmartRWMutex
	items map[string]*item
}

// New creates a new in-memory store.
func New(_ blob.Config) *Store {
	return &Store{
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "blob-memory"}),
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
		return nil, blob.ErrNotFound
	}

	return io.NopCloser(bytes.NewReader(item.data)), nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return blob.ErrNotFound
	}

	delete(s.items, key)
	return nil
}

func (s *Store) URL(key string) string {
	return "memory://" + key
}
