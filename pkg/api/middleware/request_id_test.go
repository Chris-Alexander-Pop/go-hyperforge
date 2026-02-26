package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestID_Uniqueness(t *testing.T) {
	// Create a simple handler wrapped by the middleware
	handler := RequestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First Request
	req1 := httptest.NewRequest("GET", "/", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	id1 := w1.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id1, "Request ID should not be empty")

	// Second Request
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	id2 := w2.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id2, "Request ID should not be empty")

	// Verify IDs are unique
	assert.NotEqual(t, id1, id2, "Request IDs should be unique but got identical IDs: %s", id1)
}
