package compute

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for compute. Subpackages (vm, container, serverless)
// define additional typed sentinels.
const (
	CodeInvalidArgument = "COMPUTE_INVALID_ARGUMENT"
	CodeNotFound        = "COMPUTE_NOT_FOUND"
	CodeConflict        = "COMPUTE_CONFLICT"
	CodeUnavailable     = "COMPUTE_UNAVAILABLE"
	CodeInternal        = "COMPUTE_INTERNAL"
	CodeNotSupported    = "COMPUTE_NOT_SUPPORTED"
)

// Sentinel errors shared across compute subdomains.
var (
	// ErrInvalidArgument is returned when compute input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid compute argument", nil)

	// ErrNotFound is returned when a compute resource is missing.
	ErrNotFound = errors.New(CodeNotFound, "compute resource not found", nil)

	// ErrConflict is returned for invalid instance/container state transitions.
	ErrConflict = errors.New(CodeConflict, "compute conflict", nil)

	// ErrUnavailable is returned when a compute backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "compute backend unavailable", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this compute adapter", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid compute argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected compute subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal compute error"
	}
	return errors.New(CodeInternal, msg, err)
}
