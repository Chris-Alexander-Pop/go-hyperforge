package middleware

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/api/ratelimit"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

// KeyFunc extracts a rate-limit bucket key from the request.
type KeyFunc func(r *http.Request) string

// KeyByIP uses the client IP (RemoteAddr host, without port).
func KeyByIP(r *http.Request) string {
	key, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return key
}

// KeyByUser uses the authenticated subject from AuthMiddleware context,
// falling back to IP when no subject is present.
func KeyByUser(r *http.Request) string {
	if sub := GetSubject(r.Context()); sub != "" {
		return "user:" + sub
	}
	return "ip:" + KeyByIP(r)
}

// KeyByAPIKey uses the X-API-Key header, falling back to IP when absent.
func KeyByAPIKey(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return "apikey:" + key
	}
	return "ip:" + KeyByIP(r)
}

// RateLimitMiddleware creates an HTTP handler that enforces rate limits keyed by IP.
func RateLimitMiddleware(limiter ratelimit.Limiter, limit int64, period time.Duration) func(http.Handler) http.Handler {
	return RateLimitMiddlewareWithKey(limiter, limit, period, KeyByIP)
}

// RateLimitMiddlewareWithKey creates rate-limit middleware with a custom key strategy
// (e.g. KeyByUser, KeyByAPIKey, or a composition).
func RateLimitMiddlewareWithKey(limiter ratelimit.Limiter, limit int64, period time.Duration, keyFn KeyFunc) func(http.Handler) http.Handler {
	if keyFn == nil {
		keyFn = KeyByIP
	}
	limitStr := strconv.FormatInt(limit, 10)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)

			res, err := limiter.Allow(r.Context(), key, limit, period)
			if err != nil {
				// Fail open for limiter backend errors (availability over strict limiting).
				logger.L().ErrorContext(r.Context(), "rate limit check failed", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", limitStr)
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
