package circuitbreaker

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for the mesh-facing circuit breaker facade.
// Prefer comparing with errors.Is. Codes align with pkg/resilience /
// pkg/errors (UNAVAILABLE / RESOURCE_EXHAUSTED).
var (
	// ErrCircuitOpen is returned when the circuit is open.
	ErrCircuitOpen = errors.Unavailable("circuit breaker is open", nil)

	// ErrTooManyRequests is returned when too many requests in half-open state.
	ErrTooManyRequests = errors.ResourceExhausted("too many requests in half-open state", nil)
)
