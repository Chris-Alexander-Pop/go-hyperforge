package dns

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Sentinel errors for DNS operations.
var (
	// ErrZoneNotFound is returned when a zone does not exist.
	ErrZoneNotFound = errors.NotFound("zone not found", nil)

	// ErrZoneAlreadyExists is returned when a zone already exists.
	ErrZoneAlreadyExists = errors.Conflict("zone already exists", nil)

	// ErrRecordNotFound is returned when a record does not exist.
	ErrRecordNotFound = errors.NotFound("record not found", nil)

	// ErrRecordAlreadyExists is returned when a record already exists.
	ErrRecordAlreadyExists = errors.Conflict("record already exists", nil)

	// ErrInvalidRecordType is returned for invalid record types.
	ErrInvalidRecordType = errors.InvalidArgument("invalid record type", nil)

	// ErrInvalidRecordValue is returned for invalid record values.
	ErrInvalidRecordValue = errors.InvalidArgument("invalid record value", nil)

	// ErrInvalidDomain is returned for invalid domain names.
	ErrInvalidDomain = errors.InvalidArgument("invalid domain name", nil)

	// ErrConnectionFailed is returned when the DNS provider is unreachable.
	ErrConnectionFailed = errors.Internal("DNS provider connection failed", nil)
)
