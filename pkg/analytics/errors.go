package analytics

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for analytics operations.
var (
	// ErrCounterNotFound is returned when a named counter does not exist.
	// Count does not use this sentinel: missing counters return (0, nil).
	// Merge returns it when the source counter is missing.
	ErrCounterNotFound = errors.NotFound("counter not found", nil)

	// ErrClosed is returned when operating on a closed Tracker.
	ErrClosed = errors.Unavailable("analytics tracker is closed", nil)
)

// IsNotFound reports whether err indicates a missing counter (NOT_FOUND).
func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.CodeNotFound)
}
