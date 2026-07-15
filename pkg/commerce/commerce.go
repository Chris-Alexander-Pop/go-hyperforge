package commerce

import (
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Money represents a monetary amount in minor units (e.g. cents for USD).
// Never use float64 for payment amounts — prefer Money in new APIs.
type Money struct {
	// Amount is the value in the currency's minor units (int64).
	Amount int64

	// Currency is an ISO 4217 currency code (e.g. "USD", "EUR", "JPY").
	Currency string
}

// NewMoney constructs a Money value. Currency is normalized to uppercase.
func NewMoney(amount int64, currency string) Money {
	return Money{
		Amount:   amount,
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

// Zero returns a zero-amount Money in the given currency.
func Zero(currency string) Money {
	return NewMoney(0, currency)
}

// IsZero reports whether the amount is zero (currency may still be set).
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// Equal reports whether two Money values share the same amount and currency.
func (m Money) Equal(other Money) bool {
	return m.Amount == other.Amount && strings.EqualFold(m.Currency, other.Currency)
}

// SameCurrency reports whether other uses the same currency code.
func (m Money) SameCurrency(other Money) bool {
	return strings.EqualFold(m.Currency, other.Currency)
}

// Add returns the sum of two Money values. Currencies must match.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency == "" || other.Currency == "" {
		return Money{}, errors.InvalidArgument("currency is required", nil)
	}
	if !m.SameCurrency(other) {
		return Money{}, errors.InvalidArgument(
			fmt.Sprintf("currency mismatch: %s vs %s", m.Currency, other.Currency),
			nil,
		)
	}
	return NewMoney(m.Amount+other.Amount, m.Currency), nil
}

// Sub returns m - other. Currencies must match.
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency == "" || other.Currency == "" {
		return Money{}, errors.InvalidArgument("currency is required", nil)
	}
	if !m.SameCurrency(other) {
		return Money{}, errors.InvalidArgument(
			fmt.Sprintf("currency mismatch: %s vs %s", m.Currency, other.Currency),
			nil,
		)
	}
	return NewMoney(m.Amount-other.Amount, m.Currency), nil
}

// Negate returns a Money with the opposite sign.
func (m Money) Negate() Money {
	return NewMoney(-m.Amount, m.Currency)
}

// Decimals returns the number of minor-unit digits for an ISO 4217 currency.
// Unknown currencies default to 2 (most common).
func Decimals(currency string) int {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "JPY", "KRW", "VND", "CLP", "XOF", "XAF", "XPF":
		return 0
	case "BHD", "IQD", "JOD", "KWD", "LYD", "OMR", "TND":
		return 3
	default:
		return 2
	}
}

// Format returns a human-readable representation (e.g. "USD 10.00", "JPY 1000").
// Formatting uses integer arithmetic only — no float64.
func Format(m Money) string {
	cur := strings.ToUpper(strings.TrimSpace(m.Currency))
	if cur == "" {
		cur = "XXX"
	}
	dec := Decimals(cur)
	if dec == 0 {
		return fmt.Sprintf("%s %d", cur, m.Amount)
	}

	neg := m.Amount < 0
	abs := m.Amount
	if neg {
		abs = -abs
	}

	scale := int64(1)
	for i := 0; i < dec; i++ {
		scale *= 10
	}
	major := abs / scale
	minor := abs % scale

	sign := ""
	if neg {
		sign = "-"
	}
	return fmt.Sprintf("%s%s %d.%0*d", sign, cur, major, dec, minor)
}

// String implements fmt.Stringer via Format.
func (m Money) String() string {
	return Format(m)
}

// Validate returns an error if Currency is empty.
func (m Money) Validate() error {
	if strings.TrimSpace(m.Currency) == "" {
		return errors.InvalidArgument("currency is required", nil)
	}
	return nil
}
