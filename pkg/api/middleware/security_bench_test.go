package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkSecurityHeaders benchmarks the performance of the SecurityHeaders middleware.
// Pre-calculating HSTS and CSP headers significantly reduces allocations per request.
func BenchmarkSecurityHeaders(b *testing.B) {
	cfg := DefaultSecurityHeadersConfig()
	handler := SecurityHeaders(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
