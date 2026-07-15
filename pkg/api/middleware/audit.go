package middleware

import (
	"net/http"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/audit"
)

// AuditMiddleware logs HTTP requests to the audit log.
func AuditMiddleware(auditor audit.Auditor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rec, r)

			// Log the request (audit failures must not break the response path)
			outcome := audit.OutcomeSuccess
			if rec.statusCode >= 400 {
				outcome = audit.OutcomeFailure
			}

			_ = auditor.LogWithBuilder(r.Context(), audit.EventTypeDataRead).
				Actor(GetSubject(r.Context()), "user").
				ActorIP(r.RemoteAddr).
				Action(r.Method+" "+r.URL.Path).
				Outcome(outcome).
				Metadata("status_code", rec.statusCode).
				Metadata("duration_ms", time.Since(start).Milliseconds()).
				Metadata("user_agent", r.UserAgent()).
				RequestID(r.Header.Get("X-Request-ID")).
				Send()
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
