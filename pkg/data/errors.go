package data

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for data processing. Subpackages (search, bigdata)
// define additional typed sentinels.
const (
	CodeInvalidArgument = "DATA_INVALID_ARGUMENT"
	CodeNotFound        = "DATA_NOT_FOUND"
	CodeUnavailable     = "DATA_UNAVAILABLE"
	CodeInternal        = "DATA_INTERNAL"
	CodeNotSupported    = "DATA_NOT_SUPPORTED"
)

// Sentinel errors shared across data subdomains.
var (
	// ErrInvalidArgument is returned when data input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid data argument", nil)

	// ErrNotFound is returned when a document, index, or table is missing.
	ErrNotFound = errors.New(CodeNotFound, "data resource not found", nil)

	// ErrUnavailable is returned when a search/warehouse backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "data backend unavailable", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this data adapter", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid data argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected data subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal data error"
	}
	return errors.New(CodeInternal, msg, err)
}
