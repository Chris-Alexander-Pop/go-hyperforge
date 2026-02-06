package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()
	// Ensure defaults are what we expect for the test
	cfg.HSTSEnabled = true
	cfg.HSTSMaxAge = 31536000
	cfg.HSTSIncludeSubdomains = true
	cfg.HSTSPreload = false

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check HSTS header
	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000", "HSTS max-age should be correct")
	assert.Contains(t, hsts, "includeSubDomains", "HSTS should include subdomains")
}

func TestCORSMaxAge(t *testing.T) {
	cfg := DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.MaxAge = 86400

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Check Access-Control-Max-Age header
	maxAge := w.Header().Get("Access-Control-Max-Age")
	assert.Equal(t, "86400", maxAge, "Access-Control-Max-Age should be 86400")
}
