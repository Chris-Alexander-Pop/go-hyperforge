package middleware

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// RateLimitMiddleware creates an HTTP handler that enforces rate limits
func RateLimitMiddleware(limiter ratelimit.Limiter, limit int64, period time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Key strategy: IP Based for now
			// In production, use "X-Forwarded-For" or User ID from context
			key := r.RemoteAddr
			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				key = host
			}

			res, err := limiter.Allow(r.Context(), key, limit, period)
			if err != nil {
				// Fail open or closed? usually open for cache errors, but closed for security.
				// We'll log and assume allowed to fail open (service availability > rate limit)
				logger.L().ErrorContext(r.Context(), "rate limit check failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Add Headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(res.Reset).Unix()))

			if !res.Allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
