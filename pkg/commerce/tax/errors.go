package tax

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

var (
	// ErrUnsupportedLocation indicates tax cannot be calculated for the location.
	ErrUnsupportedLocation = errors.InvalidArgument("unsupported tax location", nil)

	// ErrInvalidAmount indicates the taxable amount is invalid.
	ErrInvalidAmount = errors.InvalidArgument("invalid taxable amount", nil)

	// ErrRateNotFound indicates no rate is configured for the jurisdiction.
	ErrRateNotFound = errors.NotFound("tax rate not found for jurisdiction", nil)
)
