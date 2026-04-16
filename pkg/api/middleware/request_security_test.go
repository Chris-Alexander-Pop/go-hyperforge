package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireHTTPS(t *testing.T) {
	tests := []struct {
		name         string
		proto        string
		host         string
		allowedHosts []string
		expectedCode int
		expectedLoc  string
	}{
		{
			name:         "Already HTTPS",
			proto:        "https",
			host:         "example.com",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Redirect HTTP to HTTPS",
			proto:        "http",
			host:         "example.com",
			expectedCode: http.StatusMovedPermanently,
			expectedLoc:  "https://example.com/",
		},
		{
			name:         "Reject invalid Host character (no allowedHosts)",
			proto:        "http",
			host:         "evil.com/malicious",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Reject allowedHosts mismatch",
			proto:        "http",
			host:         "evil.com",
			allowedHosts: []string{"example.com"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Accept allowedHosts match",
			proto:        "http",
			host:         "example.com",
			allowedHosts: []string{"example.com"},
			expectedCode: http.StatusMovedPermanently,
			expectedLoc:  "https://example.com/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var handler http.Handler
			if len(tc.allowedHosts) > 0 {
				handler = RequireHTTPS(tc.allowedHosts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			} else {
				handler = RequireHTTPS()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			}

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tc.host
			req.Header.Set("X-Forwarded-Proto", tc.proto)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}

			if tc.expectedLoc != "" && w.Header().Get("Location") != tc.expectedLoc {
				t.Errorf("Expected location %s, got %s", tc.expectedLoc, w.Header().Get("Location"))
			}
		})
	}
}
