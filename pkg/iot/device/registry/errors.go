package registry

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel / helpers for device registry operations.
var (
	ErrDeviceNotFound      = errors.NotFound("device not found", nil)
	ErrDeviceAlreadyExists = errors.Conflict("device already registered", nil)
	ErrInvalidDevice       = errors.InvalidArgument("invalid device", nil)
)
