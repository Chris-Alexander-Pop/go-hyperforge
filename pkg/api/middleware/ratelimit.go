package middleware

import (
	"net/http"
	"strconv"
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

			res, err := limiter.Allow(r.Context(), key, limit, period)
			if err != nil {
				// Fail open or closed? usually open for cache errors, but closed for security.
				// We'll log and assume allowed to fail open (service availability > rate limit)
				logger.L().ErrorContext(r.Context(), "rate limit check failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			// Add Headers
			// ⚡ Bolt: Using strconv instead of fmt.Sprintf for ~3x faster integer formatting
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(res.Remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(res.Reset.Seconds())))

			if !res.Allowed {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
