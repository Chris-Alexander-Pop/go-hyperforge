package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_HSTS(t *testing.T) {
	// Setup
	cfg := middleware.DefaultSecurityHeadersConfig()
	cfg.HSTSMaxAge = 31536000 // 1 year

	handler := middleware.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify
	hsts := rec.Header().Get("Strict-Transport-Security")
	// The current implementation uses string(rune(31536000)) which produces garbage.
	// We expect "max-age=31536000; includeSubDomains"
	assert.Contains(t, hsts, "max-age=31536000", "HSTS header should contain correct max-age")
}

func TestCORS_MaxAge(t *testing.T) {
	// Setup
	cfg := middleware.DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"https://example.com"}
	cfg.MaxAge = 86400 // 24 hours

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Verify
	maxAge := rec.Header().Get("Access-Control-Max-Age")
	// The current implementation uses time.Duration(86400).String() which treats it as nanoseconds -> "86.4µs"
	// We expect "86400"
	assert.Equal(t, "86400", maxAge, "Access-Control-Max-Age should be in seconds")
}
