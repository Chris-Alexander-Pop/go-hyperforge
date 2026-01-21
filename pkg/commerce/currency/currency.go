package currency

import (
	"context"
	"time"
)

// Config holds currency configuration.
type Config struct {
	// Provider: "memory", "openexchangerates", etc.
	Provider string `env:"CURRENCY_PROVIDER" env-default:"memory"`
}

// ConversionResult represents the result of a currency conversion.
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
