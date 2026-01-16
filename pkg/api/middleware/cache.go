package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/cache"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func CacheMiddleware(c cache.Cache, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			key := fmt.Sprintf("resp:%s", r.RequestURI)

			// Try Get
			var cachedBody []byte
			if err := c.Get(r.Context(), key, &cachedBody); err == nil {
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Content-Type", "application/json") // Assumption!
				w.Write(cachedBody)
				return
			}

			// Miss - Record Response
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
				statusCode:     http.StatusOK, // Default
			}

			next.ServeHTTP(wrapper, r)

			// Cache if 200 OK
			if wrapper.statusCode == http.StatusOK {
				// Fire and forget set
				go func() {
					if err := c.Set(context.Background(), key, wrapper.body.Bytes(), ttl); err != nil {
						logger.L().Warn("failed to cache response", "key", key, "error", err)
					}
				}()
			}
		})
	}
}
