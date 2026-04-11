package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func TestSecurityHeaders_HSTS(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()
	cfg.HSTSEnabled = true
	cfg.HSTSMaxAge = 31536000 // 1 year
	cfg.HSTSIncludeSubdomains = true
	cfg.HSTSPreload = false

	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
}

func TestCORS_MaxAge(t *testing.T) {
	cfg := DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.MaxAge = 86400 // 24 hours

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	corsMaxAge := w.Header().Get("Access-Control-Max-Age")
	assert.Equal(t, "86400", corsMaxAge)
}

func TestCORS_WildcardCredentials(t *testing.T) {
	cfg := DefaultCORSConfig()
	cfg.AllowedOrigins = []string{"*"}
	cfg.AllowCredentials = true

	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	assert.Equal(t, "*", origin)

	credentials := w.Header().Get("Access-Control-Allow-Credentials")
	assert.Equal(t, "", credentials, "Credentials should not be allowed with wildcard origin")
}

func TestSanitizeMiddleware_Injection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.SanitizerConfig{
		StripHTML:  true,
		EscapeHTML: true,
	})
	handler := SanitizeMiddleware(sanitizer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name     string
		query    string
		wantCode int
	}{
		{"Valid Input", "?name=jules", http.StatusOK},
		{"SQL Injection", "?query=SELECT%20*%20FROM%20users", http.StatusBadRequest},
		{"Path Traversal", "?file=../../../etc/passwd", http.StatusBadRequest},
		{"Command Injection", "?cmd=ls%20-la%20%3B", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}
