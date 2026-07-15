package ai

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Domain-wide error codes for the AI umbrella. Subpackages may define additional sentinels.
const (
	CodeInvalidArgument = "AI_INVALID_ARGUMENT"
	CodeNotFound        = "AI_NOT_FOUND"
	CodeUnavailable     = "AI_UNAVAILABLE"
	CodeInternal        = "AI_INTERNAL"
	CodeNotSupported    = "AI_NOT_SUPPORTED"
)

// Sentinel errors shared across AI subdomains (genai, ml, perception).
var (
	// ErrInvalidArgument is returned when AI caller input is malformed.
	ErrInvalidArgument = errors.New(CodeInvalidArgument, "invalid AI argument", nil)

	// ErrNotFound is returned when an AI resource (model, job, prompt) is missing.
	ErrNotFound = errors.New(CodeNotFound, "AI resource not found", nil)

	// ErrUnavailable is returned when an AI backend is unreachable.
	ErrUnavailable = errors.New(CodeUnavailable, "AI backend unavailable", nil)

	// ErrNotSupported is returned when an adapter does not support an operation.
	ErrNotSupported = errors.New(CodeNotSupported, "operation not supported by this AI adapter", nil)
)

// ErrInvalid wraps a validation failure with a domain code.
func ErrInvalid(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "invalid AI argument"
	}
	return errors.New(CodeInvalidArgument, msg, err)
}

// ErrInternal wraps an unexpected AI subsystem failure.
func ErrInternal(msg string, err error) *errors.AppError {
	if msg == "" {
		msg = "internal AI error"
	}
	return errors.New(CodeInternal, msg, err)
}
