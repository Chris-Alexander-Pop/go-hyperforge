package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeMiddleware_CommandInjection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	handler := SanitizeMiddleware(sanitizer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Payload with command injection
	req := httptest.NewRequest("GET", "/?q=hello%3B+ls", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid input detected")
}
