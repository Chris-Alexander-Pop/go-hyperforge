package cloud

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for private-cloud IaaS. Subpackages (hypervisor, provisioning,
// scheduler, controlplane) define additional typed sentinels.
const (
	CodeInvalidArgument = "CLOUD_INVALID_ARGUMENT"
	CodeNotFound        = "CLOUD_NOT_FOUND"
	CodeConflict        = "CLOUD_CONFLICT"
	CodeUnavailable     = "CLOUD_UNAVAILABLE"
	CodeInternal        = "CLOUD_INTERNAL"
	CodeNotSupported    = "CLOUD_NOT_SUPPORTED"
)

// Sentinel errors shared across cloud subdomains.
var (
	// ErrInvalidArgument is returned when cloud input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid cloud argument", nil)

	// ErrNotFound is returned when a host, instance, or inventory record is missing.
	ErrNotFound = errors.New(CodeNotFound, "cloud resource not found", nil)

	// ErrConflict is returned for capacity or state conflicts.
	ErrConflict = errors.New(CodeConflict, "cloud conflict", nil)

	// ErrUnavailable is returned when a cloud backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "cloud backend unavailable", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this cloud adapter", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid cloud argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected cloud subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal cloud error"
	}
	return errors.New(CodeInternal, msg, err)
}
