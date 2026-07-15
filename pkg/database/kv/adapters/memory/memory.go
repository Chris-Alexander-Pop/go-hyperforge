package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type item struct {
	value     []byte
	expiresAt time.Time
}

// Adapter implements kv.KV with an in-memory store.
type Adapter struct {
	items map[string]item
	mu    *concurrency.SmartRWMutex
}

// New creates a new in-memory key-value store.
func New() *Adapter {
	return &Adapter{
		items: make(map[string]item),
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-kv"}),
	}
}

// Get retrieves a value by key.
func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	it, ok := a.items[key]
	if !ok {
		return nil, errors.NotFound("key not found", nil)
	}

	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		return nil, errors.NotFound("key expired", nil)
	}

	// Return a copy to avoid mutation
	value := make([]byte, len(it.value))
	copy(value, it.value)
	return value, nil
}

// Set stores a value with the given TTL.
func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Copy the value to avoid external mutation
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)

	it := item{value: valueCopy}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	}

	a.items[key] = it
	return nil
}

// Delete removes a key.
func (a *Adapter) Delete(ctx context.Context, key string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.items, key)
	return nil
}

// Exists checks if a key exists.
func (a *Adapter) Exists(ctx context.Context, key string) (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	it, ok := a.items[key]
	if !ok {
		return false, nil
	}

	if !it.expiresAt.IsZero() && time.Now().After(it.expiresAt) {
		return false, nil
	}

	return true, nil
}

// Close clears the in-memory store.
func (a *Adapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.items = make(map[string]item)
	return nil
}

// Ensure Adapter implements kv.KV
var _ kv.KV = (*Adapter)(nil)
