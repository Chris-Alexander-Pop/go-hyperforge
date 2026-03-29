package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func TestSanitizeMiddleware_Injection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	mw := SanitizeMiddleware(sanitizer)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	tests := []struct {
		name     string
		param    string
		wantCode int
	}{
		{
			name:     "Valid Input",
			param:    "normal_value",
			wantCode: http.StatusOK,
		},
		{
			name:     "Command Injection",
			param:    "value; ls -la",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "SQL Injection",
			param:    "value' OR 1=1--",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Path Traversal",
			param:    "../../../etc/passwd",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, _ := url.Parse("http://example.com/api?param=" + url.QueryEscape(tt.param))
			req := httptest.NewRequest("GET", u.String(), nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Errorf("expected status %d, got %d", tt.wantCode, rr.Code)
			}
		})
	}
}
