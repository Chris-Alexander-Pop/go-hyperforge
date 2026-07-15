package security

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes. Subpackages may define additional typed sentinels.
const (
	CodeInvalidArgument = "SECURITY_INVALID_ARGUMENT"
	CodeNotFound        = "SECURITY_NOT_FOUND"
	CodeUnauthorized    = "SECURITY_UNAUTHORIZED"
	CodeForbidden       = "SECURITY_FORBIDDEN"
	CodeInternal        = "SECURITY_INTERNAL"
	CodeNotSupported    = "SECURITY_NOT_SUPPORTED"
	CodeUnavailable     = "SECURITY_UNAVAILABLE"
)

// Sentinel errors shared across security subdomains.
var (
	// ErrInvalidArgument is returned when caller input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid security argument", nil)

	// ErrNotFound is returned when a security resource (secret, key, rule) is missing.
	ErrNotFound = errors.New(CodeNotFound, "security resource not found", nil)

	// ErrForbidden is returned when a security check rejects the request.
	ErrForbidden = errors.New(CodeForbidden, "security check rejected", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this security adapter", nil)

	// ErrUnavailable is returned when a remote security backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "security backend unavailable", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid security argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected security subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal security error"
	}
	return errors.New(CodeInternal, msg, err)
}
