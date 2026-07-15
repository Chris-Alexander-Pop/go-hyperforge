package enterprise

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for enterprise patterns. Subpackages (cqrs, eventsource, ddd)
// define additional typed sentinels.
const (
	CodeInvalidArgument = "ENTERPRISE_INVALID_ARGUMENT"
	CodeNotFound        = "ENTERPRISE_NOT_FOUND"
	CodeConflict        = "ENTERPRISE_CONFLICT"
	CodeUnavailable     = "ENTERPRISE_UNAVAILABLE"
	CodeInternal        = "ENTERPRISE_INTERNAL"
)

// Sentinel errors shared across enterprise subdomains.
var (
	// ErrInvalidArgument is returned when enterprise input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid enterprise argument", nil)

	// ErrNotFound is returned when an aggregate, projection, or checkpoint is missing.
	ErrNotFound = errors.New(CodeNotFound, "enterprise resource not found", nil)

	// ErrConflict is returned for concurrency/version conflicts.
	ErrConflict = errors.New(CodeConflict, "enterprise conflict", nil)

	// ErrUnavailable is returned when an enterprise store backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "enterprise backend unavailable", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid enterprise argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected enterprise subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal enterprise error"
	}
	return errors.New(CodeInternal, msg, err)
}
