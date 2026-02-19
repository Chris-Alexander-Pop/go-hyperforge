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

// MockLimiter records calls to Allow
type MockLimiter struct {
	Calls []string
}

func (m *MockLimiter) Allow(ctx context.Context, key string, limit int64, period time.Duration) (*ratelimit.Result, error) {
	m.Calls = append(m.Calls, key)
	return &ratelimit.Result{Allowed: true, Remaining: limit - 1, Reset: period}, nil
}

func TestRateLimitMiddleware_IPSpoofing(t *testing.T) {
	limiter := &MockLimiter{}
	handler := RateLimitMiddleware(limiter, 10, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		remoteAddr string
		expectedKey string
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.0.2.1:12345",
			expectedKey: "192.0.2.1",
		},
		{
			name:       "IPv4 with different port",
			remoteAddr: "192.0.2.1:54321",
			expectedKey: "192.0.2.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:12345",
			expectedKey: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Verify the last call key
			assert.NotEmpty(t, limiter.Calls)
			lastCall := limiter.Calls[len(limiter.Calls)-1]
			assert.Equal(t, tt.expectedKey, lastCall)
		})
	}
}
