package hypervisor

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrVMNotFound is returned when a requested VM ID does not exist.
	ErrVMNotFound = errors.NotFound("vm not found", nil)

	// ErrVMAlreadyExists is returned when attempting to create a VM with a name/ID that already exists.
	ErrVMAlreadyExists = errors.Conflict("vm already exists", nil)

	// ErrResourceExhausted is returned when the hypervisor lacks resources to fulfill a request.
	ErrResourceExhausted = errors.Internal("resource exhausted", nil)

	// ErrOperationFailed is a generic error for hypervisor operation failures.
	ErrOperationFailed = errors.Internal("operation failed", nil)
)
