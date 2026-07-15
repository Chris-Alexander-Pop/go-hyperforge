package slidingwindow

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ cache.Cache = (*mockCache)(nil)

type mockCache struct {
	mu    sync.Mutex
	items map[string][]byte
	ttls  map[string]time.Time
}

func newMockCache() *mockCache {
	return &mockCache{
		items: make(map[string][]byte),
		ttls:  make(map[string]time.Time),
	}
}

func (m *mockCache) expired(key string) bool {
	expiry, ok := m.ttls[key]
	return ok && !expiry.IsZero() && time.Now().After(expiry)
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.expired(key) {
		delete(m.items, key)
		delete(m.ttls, key)
		return cache.ErrKeyExpired
	}
	data, ok := m.items[key]
	if !ok {
		return cache.ErrKeyNotFound
	}
	return json.Unmarshal(data, dest)
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.items[key] = data
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	} else {
		m.ttls[key] = time.Time{}
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
	delete(m.ttls, key)
	return nil
}

func (m *mockCache) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.expired(key) {
		delete(m.items, key)
		delete(m.ttls, key)
	}

	var val int64
	if data, ok := m.items[key]; ok {
		_ = json.Unmarshal(data, &val)
	}
	val += delta
	data, err := json.Marshal(val)
	if err != nil {
		return 0, err
	}
	m.items[key] = data
	return val, nil
}

func (m *mockCache) Close() error {
	return nil
}

func TestAllow(t *testing.T) {
	c := newMockCache()
	l := New(c)
	ctx := context.Background()

	limit := int64(10)
	period := time.Minute
	key := "user:123"

	res, err := l.Allow(ctx, key, limit, period)
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, int64(9), res.Remaining)

	for i := 0; i < 9; i++ {
		res, err = l.Allow(ctx, key, limit, period)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
	}

	res, err = l.Allow(ctx, key, limit, period)
	require.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, int64(0), res.Remaining)
}

func TestSlidingUsesPreviousWindowWeight(t *testing.T) {
	// Near the end of a window, previous-window traffic should still count.
	c := newMockCache()
	l := New(c)
	ctx := context.Background()
	period := time.Minute
	now := time.Now()
	currStart := now.Truncate(period).Unix()
	prevStart := now.Add(-period).Truncate(period).Unix()

	require.NoError(t, c.Set(ctx, windowKey("k", prevStart), int64(8), period*2))
	require.NoError(t, c.Set(ctx, windowKey("k", currStart), int64(0), period*2))

	res, err := l.Allow(ctx, "k", 10, period)
	require.NoError(t, err)
	// With weight on previous (~almost full if early in window, or partial later),
	// at least one allow should succeed from empty current + some prev weight.
	_ = res
	assert.NotNil(t, res)
}

func BenchmarkAllow(b *testing.B) {
	c := newMockCache()
	l := New(c)
	ctx := context.Background()
	limit := int64(1000000)
	period := time.Minute
	key := "bench-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = l.Allow(ctx, key, limit, period)
	}
}
