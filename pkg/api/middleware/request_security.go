package middleware

import (
	"net/http"

	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/google/uuid"
)

// SanitizeMiddleware sanitizes request inputs to prevent XSS and injection attacks.
func SanitizeMiddleware(sanitizer *validator.Sanitizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sanitize query parameters
			query := r.URL.Query()
			for key, values := range query {
				for i, v := range values {
					// Check for injection attempts
					if validator.DetectSQLInjection(v) || validator.DetectPathTraversal(v) {
						http.Error(w, "Invalid input detected", http.StatusBadRequest)
						return
					}
					query[key][i] = sanitizer.Sanitize(v)
				}
			}
			r.URL.RawQuery = query.Encode()

			// Sanitize common headers that might be reflected
			if referer := r.Header.Get("Referer"); referer != "" {
				r.Header.Set("Referer", sanitizer.Sanitize(referer))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureJSONMiddleware sets secure defaults for JSON responses.
func SecureJSONMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent JSON from being interpreted as HTML
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			// Prevent MIME sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			next.ServeHTTP(w, r)
		})
	}
}

// RequireHTTPS redirects HTTP requests to HTTPS.
func RequireHTTPS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check X-Forwarded-Proto for reverse proxy setups
			proto := r.Header.Get("X-Forwarded-Proto")
			if proto == "" {
				if r.TLS != nil {
					proto = "https"
				} else {
					proto = "http"
				}
			}

			if proto != "https" {
				// Redirect to HTTPS
				https := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, https, http.StatusMovedPermanently)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDMiddleware ensures each request has a unique ID for tracing.
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
				r.Header.Set("X-Request-ID", requestID)
			}
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID creates a unique request identifier using UUID v4.
func generateRequestID() string {
	return uuid.NewString()
}
