package memory

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type item struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	items map[string]item
	mu    *concurrency.SmartRWMutex
}

func New() cache.Cache {
	return &MemoryCache{
		items: make(map[string]item),
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-cache"}),
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.items[key]
	if !ok {
		return errors.New(errors.CodeNotFound, "key not found", nil)
	}

	if time.Now().After(it.expiresAt) {
		return errors.New(errors.CodeNotFound, "key expired", nil)
	}

	return json.Unmarshal(it.value, dest)
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

	it, ok := m.items[key]
	var val int64

	if ok {
		if time.Now().After(it.expiresAt) {
			val = 0
		} else {
			_ = json.Unmarshal(it.value, &val)
		}
	}

	val += delta

	data, err := json.Marshal(val)
	if err != nil {
		return 0, err
	}

	expiry := time.Now().Add(24 * time.Hour)
	if ok && time.Now().Before(it.expiresAt) {
		expiry = it.expiresAt
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
