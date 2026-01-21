package payment

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

var (
	// ErrDeclined indicates the card was declined.
	ErrDeclined = errors.New("PAYMENT_DECLINED", "payment declined", nil)

	// ErrInsufficientFunds indicates insufficient funds.
	ErrInsufficientFunds = errors.New("INSUFFICIENT_FUNDS", "insufficient funds", nil)

	// ErrInvalidCard indicates an invalid card number or details.
	ErrInvalidCard = errors.InvalidArgument("invalid card", nil)

	// ErrExpiredCard indicates the card has expired.
	ErrExpiredCard = errors.New("CARD_EXPIRED", "card expired", nil)
)
