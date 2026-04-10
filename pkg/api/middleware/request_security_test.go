package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeMiddleware_CommandInjection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.SanitizerConfig{})
	handler := SanitizeMiddleware(sanitizer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Malicious request with command injection in query parameter
	req := httptest.NewRequest("GET", "/?cmd=ls%20-la%3B%20cat%20/etc/passwd", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should be rejected with 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input detected")
}
