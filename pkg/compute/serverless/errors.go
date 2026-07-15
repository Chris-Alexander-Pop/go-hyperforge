package serverless

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for serverless operations.
var (
	// ErrFunctionNotFound is returned when a function does not exist.
	ErrFunctionNotFound = errors.NotFound("function not found", nil)

	// ErrFunctionAlreadyExists is returned when a function already exists.
	ErrFunctionAlreadyExists = errors.Conflict("function already exists", nil)

	// ErrInvocationFailed is returned when function invocation fails.
	ErrInvocationFailed = errors.Internal("invocation failed", nil)

	// ErrInvalidRuntime is returned for unsupported runtimes.
	ErrInvalidRuntime = errors.InvalidArgument("invalid runtime", nil)

	// ErrFunctionError is returned when the function execution errors.
	ErrFunctionError = errors.Internal("function execution error", nil)
)
