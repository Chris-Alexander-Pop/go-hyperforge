package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLimiter is a mock implementation of ratelimit.Limiter
type MockLimiter struct {
	mock.Mock
}

func (m *MockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	args := m.Called(ctx, key, limit, period)
	if res := args.Get(0); res != nil {
		return res.(*ratelimit.Result), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestRateLimitMiddleware_IPSpoofing(t *testing.T) {
	// This test verifies that the middleware correctly strips the port from RemoteAddr
	// ensuring that requests from the same IP (but different ports) share the same rate limit.

	mockLimiter := new(MockLimiter)

	// Setup expectations
	// We expect the middleware to call Allow with the SAME key ("1.2.3.4") for both requests
	mockLimiter.On("Allow", mock.Anything, "1.2.3.4", int64(10), time.Minute).
		Return(&ratelimit.Result{Allowed: true, Remaining: 9, Reset: time.Minute}, nil).Times(2)

	handler := RateLimitMiddleware(mockLimiter, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request 1 from port 12345
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "1.2.3.4:12345"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Request 2 from port 54321
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "1.2.3.4:54321"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	mockLimiter.AssertExpectations(t)
}

func TestRateLimitMiddleware_ResetHeader(t *testing.T) {
	// This test checks the format of X-RateLimit-Reset header.
	// It should be a Unix timestamp.

	mockLimiter := new(MockLimiter)

	// Reset in 60 seconds
	resetDuration := 60 * time.Second

	// Expect call with IP stripped
	mockLimiter.On("Allow", mock.Anything, "1.2.3.4", int64(10), time.Minute).
		Return(&ratelimit.Result{Allowed: true, Remaining: 9, Reset: resetDuration}, nil)

	handler := RateLimitMiddleware(mockLimiter, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify X-RateLimit-Reset header
	resetHeader := w.Header().Get("X-RateLimit-Reset")

	// Check if it parses as an integer (timestamp)
	ts, err := strconv.ParseInt(resetHeader, 10, 64)
	assert.NoError(t, err, "X-RateLimit-Reset should be an integer timestamp")

	// Verify it's in the future (roughly now + 60s)
	// Allow 5 second buffer for execution time
	expected := time.Now().Add(resetDuration).Unix()
	assert.InDelta(t, expected, ts, 5, "X-RateLimit-Reset timestamp mismatch")
}
