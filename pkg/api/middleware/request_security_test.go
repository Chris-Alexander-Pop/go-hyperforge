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

	// semicolon is considered a query param separator before Go 1.17,
	// but since Go 1.17 it's dropped if unescaped unless AllowQuerySemicolons is used.
	// So let's use another payload with a pipe character or URL encode it
	req := httptest.NewRequest("GET", "/?q=hello%3Bls", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
