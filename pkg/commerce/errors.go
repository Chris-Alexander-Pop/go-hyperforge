package commerce

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for commerce. Subpackages (payment, billing, tax, currency)
// define additional typed sentinels.
const (
	CodeInvalidArgument = "COMMERCE_INVALID_ARGUMENT"
	CodeNotFound        = "COMMERCE_NOT_FOUND"
	CodeConflict        = "COMMERCE_CONFLICT"
	CodeUnavailable     = "COMMERCE_UNAVAILABLE"
	CodeInternal        = "COMMERCE_INTERNAL"
)

// Sentinel errors shared across commerce subdomains.
var (
	// ErrInvalidArgument is returned when commerce input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid commerce argument", nil)

	// ErrNotFound is returned when a commerce resource is missing.
	ErrNotFound = errors.New(CodeNotFound, "commerce resource not found", nil)

	// ErrConflict is returned for conflicting commerce state (e.g. currency mismatch).
	ErrConflict = errors.New(CodeConflict, "commerce conflict", nil)

	// ErrUnavailable is returned when a payment/tax/FX backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "commerce backend unavailable", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid commerce argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected commerce subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal commerce error"
	}
	return errors.New(CodeInternal, msg, err)
}
