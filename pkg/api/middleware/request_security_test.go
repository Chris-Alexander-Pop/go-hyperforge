package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
)

func TestSanitizeMiddleware_CommandInjection(t *testing.T) {
	sanitizer := validator.NewSanitizer(validator.SanitizerConfig{})
	mw := SanitizeMiddleware(sanitizer)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/data?q=some_user_input%3B+echo+%27hello%27", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status Bad Request for command injection, got %v", rec.Code)
	}

	reqSafe := httptest.NewRequest("GET", "/api/data?q=hello", nil)
	recSafe := httptest.NewRecorder()

	handler.ServeHTTP(recSafe, reqSafe)

	if recSafe.Code != http.StatusOK {
		t.Errorf("Expected status OK for safe input, got %v", recSafe.Code)
	}
}
