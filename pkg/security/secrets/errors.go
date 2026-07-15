package secrets

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

const (
	CodeNotFound        = "SECRET_NOT_FOUND"
	CodeInvalidArgument = "SECRET_INVALID_ARGUMENT"
	CodeRotateFailed    = "SECRET_ROTATE_FAILED"
	CodeUnavailable     = "SECRET_UNAVAILABLE"
)

var (
	// ErrNotFound is returned when a named secret does not exist.
	ErrNotFound = errors.New(CodeNotFound, "secret not found", nil)

	// ErrInvalidArgument is returned when secret name/value/config is invalid.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid secret argument", nil)

	// ErrRotateFailed is returned when rotation cannot be completed.
	ErrRotateFailed = errors.New(CodeRotateFailed, "secret rotation failed", nil)

	// ErrUnavailable is returned when a remote secrets backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "secrets backend unavailable", nil)
)
