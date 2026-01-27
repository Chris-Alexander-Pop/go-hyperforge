package archive

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for archive storage operations.
var (
	// ErrObjectNotFound is returned when an archived object does not exist.
	ErrObjectNotFound = errors.NotFound("archived object not found", nil)

	// ErrObjectNotRestored is returned when trying to download an object that hasn't been restored.
	ErrObjectNotRestored = errors.Conflict("object has not been restored", nil)

	// ErrRestoreExpired is returned when the restored copy has expired.
	ErrRestoreExpired = errors.Conflict("restored copy has expired", nil)

	// ErrRestoreInProgress is returned when a restore is already in progress for the object.
	ErrRestoreInProgress = errors.Conflict("restore already in progress", nil)

	// ErrInvalidStorageClass is returned when an invalid storage class is specified.
	ErrInvalidStorageClass = errors.InvalidArgument("invalid storage class", nil)

	// ErrInvalidRestoreTier is returned when an invalid restore tier is specified.
	ErrInvalidRestoreTier = errors.InvalidArgument("invalid restore tier", nil)

	// ErrQuotaExceeded is returned when storage quota is exceeded.
	ErrQuotaExceeded = errors.Conflict("storage quota exceeded", nil)
)
