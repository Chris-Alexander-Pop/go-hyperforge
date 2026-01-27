package loadbalancer

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for load balancer operations.
var (
	// ErrLoadBalancerNotFound is returned when a load balancer does not exist.
	ErrLoadBalancerNotFound = errors.NotFound("load balancer not found", nil)

	// ErrListenerNotFound is returned when a listener does not exist.
	ErrListenerNotFound = errors.NotFound("listener not found", nil)

	// ErrTargetPoolNotFound is returned when a target pool does not exist.
	ErrTargetPoolNotFound = errors.NotFound("target pool not found", nil)

	// ErrTargetNotFound is returned when a target does not exist.
	ErrTargetNotFound = errors.NotFound("target not found", nil)

	// ErrRuleNotFound is returned when a rule does not exist.
	ErrRuleNotFound = errors.NotFound("rule not found", nil)

	// ErrTargetAlreadyRegistered is returned when a target is already registered.
	ErrTargetAlreadyRegistered = errors.Conflict("target already registered", nil)

	// ErrInvalidProtocol is returned for invalid protocols.
	ErrInvalidProtocol = errors.InvalidArgument("invalid protocol", nil)

	// ErrInvalidPort is returned for invalid port numbers.
	ErrInvalidPort = errors.InvalidArgument("invalid port number", nil)

	// ErrLoadBalancerInUse is returned when deleting a load balancer with listeners.
	ErrLoadBalancerInUse = errors.Conflict("load balancer has active listeners", nil)

	// ErrTargetPoolInUse is returned when deleting a target pool with targets.
	ErrTargetPoolInUse = errors.Conflict("target pool has registered targets", nil)
)
