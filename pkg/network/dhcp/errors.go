package dhcp

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrPoolNotFound is returned when a requested IP pool does not exist.
	ErrPoolNotFound = errors.NotFound("ip pool not found", nil)

	// ErrIPExhausted is returned when no more IPs are available in the pool.
	ErrIPExhausted = errors.Internal("ip pool exhausted", nil)

	// ErrIPAlreadyAllocated is returned when attempting to reserve an IP that is already in use.
	ErrIPAlreadyAllocated = errors.Conflict("ip already allocated", nil)
)
