package ip

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for IP intelligence operations.
var (
	// ErrInvalidIP is returned when the IP address cannot be parsed.
	ErrInvalidIP = errors.InvalidArgument("invalid IP address", nil)

	// ErrLookupFailed is returned when a backend lookup fails.
	ErrLookupFailed = errors.Internal("IP lookup failed", nil)
)
