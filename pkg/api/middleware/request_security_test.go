package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeMiddleware(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.DefaultSanitizerConfig())
	handler := SanitizeMiddleware(sanitizer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	tests := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "ValidInput",
			query:          "name=clean+string&id=123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "SQLInjection",
			query:          "name=" + url.QueryEscape("SELECT * FROM users"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "PathTraversal",
			query:          "file=" + url.QueryEscape("../../../etc/passwd"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "CommandInjection",
			query:          "cmd=" + url.QueryEscape("cat /etc/passwd;"),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tc.query, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}
