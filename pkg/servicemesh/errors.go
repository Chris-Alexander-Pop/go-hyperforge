package servicemesh

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for service mesh helpers. Subpackages (discovery,
// circuitbreaker, ratelimit) define additional typed sentinels.
const (
	CodeInvalidArgument = "SERVICEMESH_INVALID_ARGUMENT"
	CodeNotFound        = "SERVICEMESH_NOT_FOUND"
	CodeUnavailable     = "SERVICEMESH_UNAVAILABLE"
	CodeInternal        = "SERVICEMESH_INTERNAL"
)

// Sentinel errors shared across servicemesh subdomains.
var (
	// ErrInvalidArgument is returned when mesh input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid servicemesh argument", nil)

	// ErrNotFound is returned when a service or instance is missing from discovery.
	ErrNotFound = errors.New(CodeNotFound, "servicemesh resource not found", nil)

	// ErrUnavailable is returned when a mesh backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "servicemesh backend unavailable", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid servicemesh argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected servicemesh subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal servicemesh error"
	}
	return errors.New(CodeInternal, msg, err)
}
