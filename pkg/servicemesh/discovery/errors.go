package discovery

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for service discovery operations.
var (
	// ErrServiceNotFound is returned when a service does not exist.
	ErrServiceNotFound = errors.NotFound("service not found", nil)

	// ErrServiceAlreadyExists is returned when a service already exists.
	ErrServiceAlreadyExists = errors.Conflict("service already registered", nil)

	// ErrInvalidService is returned for invalid service configuration.
	ErrInvalidService = errors.InvalidArgument("invalid service configuration", nil)

	// ErrConnectionFailed is returned when the registry is unreachable.
	ErrConnectionFailed = errors.Internal("registry connection failed", nil)

	// ErrRegistrationFailed is returned when registration fails.
	ErrRegistrationFailed = errors.Internal("service registration failed", nil)

	// ErrWatchClosed is returned when a watch channel is closed.
	ErrWatchClosed = errors.Internal("watch channel closed", nil)
)
