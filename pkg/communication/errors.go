package communication

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors shared across communication channels.
var (
	// ErrInvalidDriver is returned when an unsupported driver is specified.
	ErrInvalidDriver = errors.InvalidArgument("invalid communication driver", nil)

	// ErrInvalidConfig is returned when configuration fails validation.
	ErrInvalidConfig = errors.InvalidArgument("invalid communication configuration", nil)

	// ErrSendFailed is returned when a message could not be delivered.
	ErrSendFailed = errors.Internal("failed to send message", nil)

	// ErrTemplateNotFound is returned when a named template does not exist.
	ErrTemplateNotFound = errors.NotFound("template not found", nil)

	// ErrRenderFailed is returned when template rendering fails.
	ErrRenderFailed = errors.Internal("failed to render template", nil)

	// ErrClosed is returned when operating on a closed sender or engine.
	ErrClosed = errors.Unavailable("communication client is closed", nil)
)

// IsNotFound reports whether err indicates a missing resource (NOT_FOUND).
func IsNotFound(err error) bool {
	return errors.IsCode(err, errors.CodeNotFound)
}

// IsInvalidArgument reports whether err indicates invalid input.
func IsInvalidArgument(err error) bool {
	return errors.IsCode(err, errors.CodeInvalidArgument)
}
