package currency

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
)

// Config holds currency configuration.
type Config struct {
	// Provider: "memory", "openexchangerates", etc.
	Provider string `env:"CURRENCY_PROVIDER" env-default:"memory"`
}

// ConversionResult represents the result of a currency conversion.
// Rates remain float64 (FX feeds are floating-point); payment amounts should
// use commerce.Money separately.
type ConversionResult struct {
	FromAmount float64
	From       string
	ToAmount   float64
	To         string
	Rate       float64
	Timestamp  time.Time
}

// Converter defines the currency conversion interface.
type Converter interface {
	// Convert converts an amount from one currency to another.
	Convert(ctx context.Context, amount float64, from string, to string) (*ConversionResult, error)

	// GetRate returns the exchange rate between two currencies.
	GetRate(ctx context.Context, from string, to string) (float64, error)
}

// LiveRateProvider optionally fetches live FX rates from an external feed.
// Memory adapters keep a static table; live providers implement this to refresh.
type LiveRateProvider interface {
	// FetchRates returns rates keyed by currency code relative to base.
	FetchRates(ctx context.Context, base string) (map[string]float64, error)
}

// FormatAmount formats a major-unit float for display. Prefer commerce.Format
// with commerce.Money for payment amounts (integer minor units, no float64).
func FormatAmount(amount float64, currency string) string {
	// Bridge for FX display only — payment paths should use commerce.Money.
	dec := commerce.Decimals(currency)
	scale := 1.0
	for i := 0; i < dec; i++ {
		scale *= 10
	}
	minor := int64(amount*scale + 0.5)
	if amount < 0 {
		minor = int64(amount*scale - 0.5)
	}
	return commerce.Format(commerce.NewMoney(minor, currency))
}

// FormatMoney formats a commerce.Money value (preferred).
func FormatMoney(m commerce.Money) string {
	return commerce.Format(m)
}
