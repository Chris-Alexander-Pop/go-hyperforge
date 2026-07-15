package circuitbreaker

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for the mesh-facing circuit breaker facade.
// Prefer comparing with errors.Is. Domain codes stay local until pkg/errors
// gains UNAVAILABLE / RESOURCE_EXHAUSTED mappings.
var (
	// ErrCircuitOpen is returned when the circuit is open.
	ErrCircuitOpen = errors.Conflict("circuit breaker is open", nil)

	// ErrTooManyRequests is returned when too many requests in half-open state.
	ErrTooManyRequests = errors.Conflict("too many requests in half-open state", nil)
)
