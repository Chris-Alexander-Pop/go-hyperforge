package storage

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for storage. Subpackages (blob, file, block, archive, controller)
// define additional typed sentinels.
const (
	CodeInvalidArgument = "STORAGE_INVALID_ARGUMENT"
	CodeNotFound        = "STORAGE_NOT_FOUND"
	CodeUnavailable     = "STORAGE_UNAVAILABLE"
	CodeInternal        = "STORAGE_INTERNAL"
	CodeNotSupported    = "STORAGE_NOT_SUPPORTED"
)

// Sentinel errors shared across storage subdomains.
var (
	// ErrInvalidArgument is returned when storage input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid storage argument", nil)

	// ErrNotFound is returned when a storage object or volume is missing.
	ErrNotFound = errors.New(CodeNotFound, "storage resource not found", nil)

	// ErrUnavailable is returned when a storage backend is unreachable or closed.
	ErrUnavailable = errors.New(CodeUnavailable, "storage backend unavailable", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this storage adapter", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid storage argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected storage subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal storage error"
	}
	return errors.New(CodeInternal, msg, err)
}
