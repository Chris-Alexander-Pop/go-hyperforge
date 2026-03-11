package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
)

type nopLimiter struct{}

func (n *nopLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	return &ratelimit.Result{Allowed: true, Remaining: 10, Reset: time.Minute}, nil
}

func BenchmarkRateLimitMiddleware(b *testing.B) {
	limiter := &nopLimiter{}
	handler := RateLimitMiddleware(limiter, 100, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ServeHTTP(w, req)
	}
}
