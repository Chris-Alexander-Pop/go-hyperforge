package container

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for container operations.
var (
	// ErrContainerNotFound is returned when a container does not exist.
	ErrContainerNotFound = errors.NotFound("container not found", nil)

	// ErrImageNotFound is returned when an image does not exist.
	ErrImageNotFound = errors.NotFound("image not found", nil)

	// ErrContainerNotRunning is returned for operations requiring a running container.
	ErrContainerNotRunning = errors.Conflict("container is not running", nil)

	// ErrContainerAlreadyRunning is returned when starting an already running container.
	ErrContainerAlreadyRunning = errors.Conflict("container is already running", nil)

	// ErrInvalidConfig is returned for invalid container configuration.
	ErrInvalidConfig = errors.InvalidArgument("invalid container configuration", nil)

	// ErrNameConflict is returned when a container name is already in use.
	ErrNameConflict = errors.Conflict("container name already in use", nil)
)
