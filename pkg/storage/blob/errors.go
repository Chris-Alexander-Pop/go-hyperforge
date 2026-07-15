package blob

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for blob storage operations.
var (
	// ErrNotFound is returned when a blob key does not exist.
	ErrNotFound = errors.NotFound("blob not found", nil)

	// ErrInvalidConfig is returned when blob configuration fails validation.
	ErrInvalidConfig = errors.InvalidArgument("invalid blob configuration", nil)

	// ErrClosed is returned when operating on a closed store.
	ErrClosed = errors.Unavailable("blob store is closed", nil)
)

// IsNotFound reports whether err indicates a missing blob (NOT_FOUND).
func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.CodeNotFound)
}

// IsInvalidArgument reports whether err indicates invalid input.
func IsInvalidArgument(err error) bool {
	return errors.IsCode(err, errors.CodeInvalidArgument)
}
