package vm

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for VM operations.
var (
	// ErrInstanceNotFound is returned when an instance does not exist.
	ErrInstanceNotFound = errors.NotFound("instance not found", nil)

	// ErrInvalidState is returned when an operation cannot be performed in current state.
	ErrInvalidState = errors.Conflict("invalid instance state for operation", nil)

	// ErrInvalidInstanceType is returned for unsupported instance types.
	ErrInvalidInstanceType = errors.InvalidArgument("invalid instance type", nil)

	// ErrImageNotFound is returned when an image does not exist.
	ErrImageNotFound = errors.NotFound("image not found", nil)

	// ErrQuotaExceeded is returned when instance quota is exceeded.
	ErrQuotaExceeded = errors.Conflict("instance quota exceeded", nil)

	// ErrSubnetNotFound is returned when a subnet does not exist.
	ErrSubnetNotFound = errors.NotFound("subnet not found", nil)
)
