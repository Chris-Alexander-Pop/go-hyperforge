package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_HSTS_Header(t *testing.T) {
	// Reproduce bug: HSTS header is malformed because it uses string(rune(int))
	cfg := middleware.SecurityHeadersConfig{
		HSTSEnabled:           true,
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
	}

	handler := middleware.SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "https://example.com", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	// We expect "max-age=31536000; includeSubDomains"
	// Due to the bug, it will likely be garbage or missing the number.
	// We assert the CORRECT behavior here, so the test fails until fixed.
	assert.Equal(t, "max-age=31536000; includeSubDomains", hsts)
}

func TestCORS_MaxAge_Header(t *testing.T) {
	// Reproduce bug: Access-Control-Max-Age uses time.Duration().String() which includes units (e.g. "86.4µs")
	// instead of seconds as integer.
	cfg := middleware.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		MaxAge:         86400, // 24 hours (seconds)
	}

	handler := middleware.CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "https://example.com", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	maxAge := w.Header().Get("Access-Control-Max-Age")
	// We expect "86400"
	// Due to the bug, it will likely be "86.4µs" (if interpreted as ns) or something else.
	assert.Equal(t, "86400", maxAge)
}
