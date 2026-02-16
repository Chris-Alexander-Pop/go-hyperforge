package slidingwindow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/stretchr/testify/assert"
)

// Ensure mockCache implements cache.Cache
var _ cache.Cache = (*mockCache)(nil)

// mockCache implements cache.Cache for testing.
type mockCache struct {
	mu    sync.Mutex
	items map[string]int64
	ttls  map[string]time.Time
}

func newMockCache() *mockCache {
	return &mockCache{
		items: make(map[string]int64),
		ttls:  make(map[string]time.Time),
	}
}

func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	return nil
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttls[key] = time.Now().Add(ttl)
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

	if expiry, ok := m.ttls[key]; ok && time.Now().After(expiry) {
		delete(m.items, key)
		delete(m.ttls, key)
	}

	m.items[key] += delta
	return m.items[key], nil
}

func (m *mockCache) Close() error {
	return nil
}

func TestAllow(t *testing.T) {
	c := newMockCache()
	l := New(c)
	ctx := context.Background()

	// Test case 1: Allow 10 requests per minute
	limit := int64(10)
	period := time.Minute
	key := "user:123"

	// First request
	res, err := l.Allow(ctx, key, limit, period)
	assert.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, int64(9), res.Remaining)

	// Consume remaining
	for i := 0; i < 9; i++ {
		res, err = l.Allow(ctx, key, limit, period)
		assert.NoError(t, err)
		assert.True(t, res.Allowed)
	}

	// Next request should be blocked
	res, err = l.Allow(ctx, key, limit, period)
	assert.NoError(t, err)
	assert.False(t, res.Allowed)
	assert.Equal(t, int64(0), res.Remaining)
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
