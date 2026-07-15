package currency

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

var (
	// ErrUnsupportedCurrency indicates a currency code is not in the rate table.
	ErrUnsupportedCurrency = errors.NotFound("currency not supported", nil)

	// ErrSameCurrency indicates from and to currencies are identical.
	ErrSameCurrency = errors.InvalidArgument("from and to currencies are the same", nil)

	// ErrInvalidAmount indicates a non-finite or negative conversion amount where disallowed.
	ErrInvalidAmount = errors.InvalidArgument("invalid amount", nil)

	// ErrLiveRatesUnavailable indicates a live FX provider failed.
	ErrLiveRatesUnavailable = errors.Unavailable("live exchange rates unavailable", nil)
)
