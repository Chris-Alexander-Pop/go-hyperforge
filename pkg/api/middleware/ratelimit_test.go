package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/algorithms/ratelimit"
	"github.com/stretchr/testify/assert"
)

type mockLimiter struct {
	calls []string
}

func (m *mockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	m.calls = append(m.calls, key)
	return &ratelimit.Result{
		Allowed:   true,
		Remaining: 10,
		Reset:     time.Second,
	}, nil
}

func TestRateLimitMiddleware_IPSpoofing(t *testing.T) {
	limiter := &mockLimiter{}
	mw := RateLimitMiddleware(limiter, 10, time.Minute)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1 from IP:Port1
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Request 2 from IP:Port2 (same IP, different port)
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:54321"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	// Verify that the keys passed to the limiter were the same (IP only)
	assert.Equal(t, 2, len(limiter.calls))
	if len(limiter.calls) == 2 {
		assert.Equal(t, "192.168.1.1", limiter.calls[0])
		assert.Equal(t, "192.168.1.1", limiter.calls[1])
		assert.Equal(t, limiter.calls[0], limiter.calls[1], "Keys should be the same, port must be stripped")
	}
}
