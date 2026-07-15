package workflow

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for workflow operations.
var (
	// ErrWorkflowNotFound is returned when a workflow does not exist.
	ErrWorkflowNotFound = errors.NotFound("workflow not found", nil)

	// ErrExecutionNotFound is returned when an execution does not exist.
	ErrExecutionNotFound = errors.NotFound("execution not found", nil)

	// ErrExecutionAlreadyExists is returned for duplicate execution IDs.
	ErrExecutionAlreadyExists = errors.Conflict("execution already exists", nil)

	// ErrExecutionNotRunning is returned when operation requires running execution.
	ErrExecutionNotRunning = errors.Conflict("execution is not running", nil)

	// ErrInvalidWorkflow is returned for invalid workflow definitions.
	ErrInvalidWorkflow = errors.InvalidArgument("invalid workflow definition", nil)

	// ErrExecutionFailed is returned when an execution fails.
	ErrExecutionFailed = errors.Internal("execution failed", nil)

	// ErrExecutionTimeout is returned when an execution times out.
	ErrExecutionTimeout = errors.Internal("execution timed out", nil)
)
