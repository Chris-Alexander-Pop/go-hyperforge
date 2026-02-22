package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/stretchr/testify/assert"
)

type mockLimiter struct {
	calls []string
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	m.calls = append(m.calls, key)
	return &ratelimit.Result{Allowed: true, Remaining: 10, Reset: time.Minute}, nil
}

func TestRateLimitMiddleware_IPSpoofing(t *testing.T) {
	limiter := &mockLimiter{}
	handler := RateLimitMiddleware(limiter, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.0.2.1:12345"
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.0.2.1:67890"
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	assert.Equal(t, 2, len(limiter.calls))
	// This assertion will fail if the middleware uses the full RemoteAddr including port
	assert.Equal(t, limiter.calls[0], limiter.calls[1], "Rate limiter keys should be identical (IP only), ignoring source port")
}
