package memory

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type item struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	items map[string]item
	mu    sync.RWMutex
}

func New() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]item),
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.items[key]
	if !ok {
		return errors.New(errors.CodeNotFound, "key not found", nil)
	}

	if time.Now().After(item.expiresAt) {
		// Lazy delete? Cannot modify under RLock. Just return NotFound.
		return errors.New(errors.CodeNotFound, "key expired", nil)
	}

	return json.Unmarshal(item.value, dest)
}

func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	m.items[key] = item{
		value:     data,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	item, ok := m.items[key]
	var val int64

	if ok {
		// Check expiry
		if time.Now().After(item.expiresAt) {
			val = 0
		} else {
			// Unmarshal
			_ = json.Unmarshal(item.value, &val)
		}
	}

	val += delta

	data, err := json.Marshal(val)
	if err != nil {
		return 0, err
	}

	// Incr usually preserves existing TTL or sets no TTL?
	// Redis INCR preserves TTL. If key is new, it needs a TTL or infinite?
	// For simplicity, if new, we set default infinite (or very long).
	// If existing, preserve expiry.

	expiry := time.Now().Add(24 * time.Hour) // Default for new keys
	if ok && time.Now().Before(item.expiresAt) {
		expiry = item.expiresAt
	}

	m.items[key] = item{
		value:     data,
		expiresAt: expiry,
	}

	return val, nil
}

func (m *MemoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]item)
	return nil
}
