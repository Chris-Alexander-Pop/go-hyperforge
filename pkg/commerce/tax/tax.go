package tax

import (
	"context"
)

// Config holds tax configuration.
type Config struct {
	// Provider: "memory", "taxjar", etc.
	Provider string `env:"TAX_PROVIDER" env-default:"memory"`
}

// Location represents a physical location for tax calculation.
type Location struct {
	Country    string
	State      string
	City       string
	PostalCode string
}

// TaxResult represents the calculated tax.
type TaxResult struct {
	TotalTax      float64
	Rate          float64
	Breakdown     map[string]float64 // e.g., "state": 5.0, "city": 1.0
	TaxableAmount float64
}

// Calculator defines the tax calculation interface.
type Calculator interface {
	// CalculateTax calculates tax for a given amount and location.
	CalculateTax(ctx context.Context, amount float64, loc Location) (*TaxResult, error)
}
