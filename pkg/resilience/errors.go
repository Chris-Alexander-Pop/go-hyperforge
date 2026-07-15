package resilience

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain error codes for resilience patterns.
//
// Mapping:
//   - Circuit open / half-open rejection → UNAVAILABLE (HTTP 503 / gRPC Unavailable)
//   - Bulkhead full / half-open probe cap → RESOURCE_EXHAUSTED (HTTP 429 / gRPC ResourceExhausted)
const (
	CodeCircuitOpen      = errors.CodeUnavailable
	CodeTooManyRequests  = errors.CodeResourceExhausted
	CodeBulkheadFull     = errors.CodeResourceExhausted
	CodeDeadlineExceeded = errors.CodeDeadlineExceeded
)

// ErrCircuitOpen is returned when the circuit breaker is open and rejects work.
var ErrCircuitOpen = errors.Unavailable("circuit breaker is open", nil)

// ErrTooManyRequests is returned when half-open MaxRequests probes are exhausted.
var ErrTooManyRequests = errors.ResourceExhausted("too many requests in half-open state", nil)

// ErrBulkheadFull is returned when a bulkhead cannot acquire a concurrency slot.
var ErrBulkheadFull = errors.ResourceExhausted("bulkhead concurrency limit reached", nil)
