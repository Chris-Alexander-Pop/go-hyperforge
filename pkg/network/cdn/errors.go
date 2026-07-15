package cdn

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for CDN operations.
var (
	// ErrDistributionNotFound is returned when a distribution does not exist.
	ErrDistributionNotFound = errors.NotFound("distribution not found", nil)

	// ErrInvalidationNotFound is returned when an invalidation does not exist.
	ErrInvalidationNotFound = errors.NotFound("invalidation not found", nil)

	// ErrInvalidOrigin is returned for an empty or invalid origin domain.
	ErrInvalidOrigin = errors.InvalidArgument("invalid origin domain", nil)
)
