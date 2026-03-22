package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeMiddleware_CommandInjection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	middleware := SanitizeMiddleware(sanitizer)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Valid input
	req := httptest.NewRequest("GET", "/api?name=john", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Command injection payload
	query := url.Values{}
	query.Add("name", "john; cat /etc/passwd")
	req = httptest.NewRequest("GET", "/api?"+query.Encode(), nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
