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

// MockLimiter records keys passed to Allow
type MockLimiter struct {
	Keys []string
}

func (m *MockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	m.Keys = append(m.Keys, key)
	return &ratelimit.Result{Allowed: true, Remaining: 10, Reset: time.Minute}, nil
}

func TestRateLimitMiddleware_IPSpoofing(t *testing.T) {
	mockLimiter := &MockLimiter{}
	handler := RateLimitMiddleware(mockLimiter, 100, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1: Port 12345
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	// Request 2: Port 54321 (Same IP)
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:54321"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	// Assertions
	assert.Equal(t, 2, len(mockLimiter.Keys))

	// Ensure that both keys are identical (just the IP)
	if len(mockLimiter.Keys) >= 2 {
		assert.Equal(t, "192.168.1.1", mockLimiter.Keys[0], "First key should be IP only")
		assert.Equal(t, "192.168.1.1", mockLimiter.Keys[1], "Second key should be IP only")
		assert.Equal(t, mockLimiter.Keys[0], mockLimiter.Keys[1], "Keys should be identical for same IP")
	}
}
