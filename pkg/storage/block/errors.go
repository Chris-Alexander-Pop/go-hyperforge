package block

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for block storage operations.
var (
	// ErrVolumeNotFound is returned when a volume does not exist.
	ErrVolumeNotFound = errors.NotFound("volume not found", nil)

	// ErrSnapshotNotFound is returned when a snapshot does not exist.
	ErrSnapshotNotFound = errors.NotFound("snapshot not found", nil)

	// ErrVolumeInUse is returned when attempting to delete an attached volume.
	ErrVolumeInUse = errors.Conflict("volume is attached to an instance", nil)

	// ErrVolumeNotAttached is returned when detaching a volume that isn't attached.
	ErrVolumeNotAttached = errors.InvalidArgument("volume is not attached to instance", nil)

	// ErrVolumeLimitExceeded is returned when volume quota is exceeded.
	ErrVolumeLimitExceeded = errors.Conflict("volume limit exceeded", nil)

	// ErrInvalidSize is returned when the requested size is invalid.
	ErrInvalidSize = errors.InvalidArgument("invalid volume size", nil)

	// ErrSizeTooSmall is returned when trying to shrink a volume.
	ErrSizeTooSmall = errors.InvalidArgument("new size must be >= current size", nil)

	// ErrSnapshotInProgress is returned when a snapshot is still being created.
	ErrSnapshotInProgress = errors.Conflict("snapshot creation in progress", nil)
)
