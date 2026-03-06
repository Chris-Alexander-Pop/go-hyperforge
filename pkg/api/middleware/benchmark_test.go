package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type mockLimiter struct {
	allowed bool
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	return &ratelimit.Result{
		Allowed:   m.allowed,
		Remaining: 10,
		Reset:     time.Hour,
	}, nil
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	limiter := &mockLimiter{allowed: true}
	middleware := RateLimitMiddleware(limiter, 100, time.Minute)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

type mockCache struct {
	data map[string][]byte
}
func (m *mockCache) Get(ctx context.Context, key string, dest interface{}) error {
	return errors.NotFound("not found", nil)
}
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return nil
}
func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}
func (m *mockCache) Close() error {
    return nil
}
func (m *mockCache) Incr(ctx context.Context, key string, value int64) (int64, error) {
    return 0, nil
}

func BenchmarkCacheMiddleware(b *testing.B) {
	c := &mockCache{data: make(map[string][]byte)}
	middleware := CacheMiddleware(c, time.Minute)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/v1/users", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
