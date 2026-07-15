package scheduler

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrNoHostFound depends on why no host was found (resource exhaustion or constraint satisfaction).
	ErrNoHostFound = errors.NotFound("no suitable host found", nil)

	// ErrInvalidRequirement should be returned when the request asks for impossible resources.
	ErrInvalidRequirement = errors.InvalidArgument("invalid requirements", nil)
)
