package resilience

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Domain error codes for resilience patterns.
//
// Mapping notes:
//   - pkg/errors does not yet define CodeUnavailable / CodeResourceExhausted.
//     When those codes land, CIRCUIT_OPEN should map to UNAVAILABLE (HTTP 503 /
//     gRPC Unavailable) and BULKHEAD_FULL to RESOURCE_EXHAUSTED (HTTP 429 /
//     gRPC ResourceExhausted). Until then these remain package-local codes.
const (
	CodeCircuitOpen  = "CIRCUIT_OPEN"
	CodeBulkheadFull = "BULKHEAD_FULL"
)

// ErrCircuitOpen is returned when the circuit breaker is open and rejects work.
var ErrCircuitOpen = errors.New(CodeCircuitOpen, "circuit breaker is open", nil)

// ErrBulkheadFull is returned when a bulkhead cannot acquire a concurrency slot.
var ErrBulkheadFull = errors.New(CodeBulkheadFull, "bulkhead concurrency limit reached", nil)
