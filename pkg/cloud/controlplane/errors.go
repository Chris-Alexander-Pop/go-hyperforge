package controlplane

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrHostNotFound is returned when a requested host ID is not registered.
	ErrHostNotFound = errors.NotFound("host not found", nil)

	// ErrHostAlreadyRegistered is returned when attempting to register a host ID that already exists.
	ErrHostAlreadyRegistered = errors.Conflict("host already registered", nil)

	// ErrInstanceNotFound is returned when an instance ID is unknown.
	ErrInstanceNotFound = errors.NotFound("instance not found", nil)

	// ErrInstanceAlreadyExists is returned when creating a duplicate instance name on a host.
	ErrInstanceAlreadyExists = errors.Conflict("instance already exists", nil)

	// ErrHostCapacityExhausted is returned when a host cannot accept the instance resources.
	ErrHostCapacityExhausted = errors.ResourceExhausted("host capacity exhausted", nil)

	// ErrInstanceAlreadyBound is returned when binding an instance that already has a host.
	ErrInstanceAlreadyBound = errors.Conflict("instance already bound to a host", nil)

	// ErrInstanceNotBound is returned when unbinding an unbound instance.
	ErrInstanceNotBound = errors.Conflict("instance is not bound to a host", nil)

	// ErrHostHasInstances is returned when deregistering a host that still has bound instances.
	ErrHostHasInstances = errors.Conflict("host still has bound instances", nil)

	// ErrHostNotReady is returned when binding to a host that is not ready.
	ErrHostNotReady = errors.Conflict("host not ready", nil)
)
