package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_HSTS_Format(t *testing.T) {
	// Setup
	cfg := middleware.DefaultSecurityHeadersConfig()
	cfg.HSTSEnabled = true
	cfg.HSTSMaxAge = 31536000 // 1 year in seconds
	cfg.HSTSIncludeSubdomains = true
	cfg.HSTSPreload = false

	handler := middleware.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)
	hsts := rec.Header().Get("Strict-Transport-Security")

	// The bug produces garbage output, so we expect this to fail
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")

	// Verify it doesn't contain garbage characters (rudimentary check)
	// If it was garbage, it wouldn't match "max-age=31536000"
}

func TestCORS_MaxAge_Format(t *testing.T) {
	// Setup
	cfg := middleware.DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.MaxAge = 86400 // 24 hours in seconds

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, rec.Code)
	maxAge := rec.Header().Get("Access-Control-Max-Age")

	// The bug produces "86.4µs" or "24h0m0s" depending on interpretation,
	// but spec requires seconds as integer
	assert.Equal(t, "86400", maxAge)
}
