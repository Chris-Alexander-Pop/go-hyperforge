package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
)

type mockLimiter struct{}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	return &ratelimit.Result{
		Allowed:   true,
		Remaining: limit - 1,
		Reset:     period,
	}, nil
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	limiter := &mockLimiter{}
	handler := RateLimitMiddleware(limiter, 100, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

type mockCache struct {
	cache.Cache
}

func (m *mockCache) Get(ctx context.Context, key string, value interface{}) error {
	// Simulate cache miss
	return context.DeadlineExceeded
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}

func BenchmarkCacheMiddleware(b *testing.B) {
	c := &mockCache{}
	handler := CacheMiddleware(c, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
