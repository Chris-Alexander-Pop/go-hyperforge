package memory

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

func init() {
	cache.RegisterDriver("memory", func(cfg cache.Config) (cache.Cache, error) {
		return New(), nil
	})
}

type item struct {
	value     []byte
	expiresAt time.Time // zero means no expiration
}

func (it item) expired() bool {
	return !it.expiresAt.IsZero() && time.Now().After(it.expiresAt)
}

// MemoryCache is an in-process cache.Cache implementation.
type MemoryCache struct {
	items map[string]item
	mu    *concurrency.SmartRWMutex
}

// New creates an empty in-memory cache.
func New() cache.Cache {
	return &MemoryCache{
		items: make(map[string]item),
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-cache"}),
	}
}

var (
	_ cache.Cache         = (*MemoryCache)(nil)
	_ cache.PrefixDeleter = (*MemoryCache)(nil)
)

func (m *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.items[key]
	if !ok {
		return cache.ErrKeyNotFound
	}
	if it.expired() {
		return cache.ErrKeyExpired
	}
	return json.Unmarshal(it.value, dest)
}

func (m *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	m.items[key] = item{value: data, expiresAt: expiresAt}
	return nil
}

func (m *MemoryCache) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	return nil
}

func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.items[key]
	if !ok || it.expired() {
		return false, nil
	}
	return true, nil
}

func (m *MemoryCache) MGet(ctx context.Context, keys []string, dest interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rv, err := mapDest(dest)
	if err != nil {
		return err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	elemType := rv.Type().Elem()
	for _, key := range keys {
		it, ok := m.items[key]
		if !ok || it.expired() {
			continue
		}
		ptr := reflect.New(elemType)
		if err := json.Unmarshal(it.value, ptr.Interface()); err != nil {
			return errors.Wrap(err, "failed to unmarshal mget value")
		}
		rv.SetMapIndex(reflect.ValueOf(key), ptr.Elem())
	}
	return nil
}

func (m *MemoryCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		m.items[key] = item{value: data, expiresAt: expiresAt}
	}
	return nil
}

func (m *MemoryCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	it, ok := m.items[key]
	if !ok || it.expired() {
		return cache.ErrKeyNotFound
	}
	if ttl > 0 {
		it.expiresAt = time.Now().Add(ttl)
	} else {
		it.expiresAt = time.Time{}
	}
	m.items[key] = it
	return nil
}

func (m *MemoryCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	it, ok := m.items[key]
	if !ok || it.expired() {
		return 0, cache.ErrKeyNotFound
	}
	if it.expiresAt.IsZero() {
		return -1, nil
	}
	remaining := time.Until(it.expiresAt)
	if remaining < 0 {
		return 0, cache.ErrKeyExpired
	}
	return remaining, nil
}

func (m *MemoryCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	it, ok := m.items[key]
	var val int64

	if ok {
		if it.expired() {
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
	if ok && !it.expired() {
		expiry = it.expiresAt // preserves zero = no expiration
	}

	m.items[key] = item{value: data, expiresAt: expiry}
	return val, nil
}

func (m *MemoryCache) DeletePrefix(ctx context.Context, prefix string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var n int64
	for key := range m.items {
		if strings.HasPrefix(key, prefix) {
			delete(m.items, key)
			n++
		}
	}
	return n, nil
}

func (m *MemoryCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]item)
	return nil
}

func mapDest(dest interface{}) (reflect.Value, error) {
	if dest == nil {
		return reflect.Value{}, errors.InvalidArgument("mget dest is nil", nil)
	}
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return reflect.Value{}, errors.InvalidArgument("mget dest must be a non-nil pointer to map[string]T", nil)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, errors.InvalidArgument("mget dest must be *map[string]T", nil)
	}
	if rv.IsNil() {
		rv.Set(reflect.MakeMap(rv.Type()))
	}
	return rv, nil
}
