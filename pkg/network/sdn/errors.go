package sdn

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrNetworkNotFound is returned when a requested network ID does not exist.
	ErrNetworkNotFound = errors.NotFound("network not found", nil)

	// ErrSubnetNotFound is returned when a requested subnet ID does not exist.
	ErrSubnetNotFound = errors.NotFound("subnet not found", nil)

	// ErrNetworkAlreadyExists is returned when attempting to create a network with a name/CIDR that conflicts.
	ErrNetworkAlreadyExists = errors.Conflict("network already exists", nil)

	// ErrSubnetOverlap is returned when a new subnet overlaps with an existing one.
	ErrSubnetOverlap = errors.InvalidArgument("subnet overlaps with existing subnet", nil)
)
