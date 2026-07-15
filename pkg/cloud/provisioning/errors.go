package provisioning

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrHostNotFound is returned when a requested host ID does not exist in the inventory.
	ErrHostNotFound = errors.NotFound("host not found", nil)

	// ErrHostNotReady is returned when a host is not in a state to accept provisioning commands.
	ErrHostNotReady = errors.Conflict("host not ready", nil)

	// ErrProvisioningFailed is a generic error for provisioning failures.
	ErrProvisioningFailed = errors.Internal("provisioning failed", nil)
)
