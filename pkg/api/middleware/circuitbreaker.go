package middleware

import (
	"context"
	"net/http"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// CircuitBreakerMiddleware wraps downstream handlers with circuit breaker protection.
// Use this when your handler calls external services that may fail.
func CircuitBreakerMiddleware(cb *resilience.CircuitBreaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := cb.Execute(r.Context(), func(ctx context.Context) error {
				// Create a response recorder to capture the status code
				rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
				next.ServeHTTP(rec, r.WithContext(ctx))

				// Count 5xx errors as failures
				if rec.statusCode >= 500 {
					return &serverError{statusCode: rec.statusCode}
				}
				return nil
			})

			if err == resilience.ErrCircuitOpen {
				http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
		})
	}
}

type serverError struct {
	statusCode int
}

func (e *serverError) Error() string {
	return "server error"
}
