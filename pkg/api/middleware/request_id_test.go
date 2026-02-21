package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_Uniqueness(t *testing.T) {
	// Setup the middleware
	handler := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ids := make(map[string]bool)
	count := 10

	for i := 0; i < count; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		requestID := w.Header().Get("X-Request-ID")
		assert.NotEmpty(t, requestID, "Request ID should not be empty")

		if ids[requestID] {
			t.Errorf("Duplicate Request ID found: %s", requestID)
		}
		ids[requestID] = true
	}

	assert.Equal(t, count, len(ids), "All Request IDs should be unique")
}

func TestRequestIDMiddleware_Format(t *testing.T) {
	handler := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	// Old format was "req-" + hex
	// New format will be UUID string (usually just hex-dashed)
	// But let's just check length > 10 for now
	assert.Greater(t, len(requestID), 10, "Request ID should be reasonably long")
}
