package tax

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
)

// Config holds tax configuration.
type Config struct {
	// Provider: "memory", "taxjar", "avalara".
	Provider string `env:"TAX_PROVIDER" env-default:"memory"`
}

// Location represents a physical location for tax calculation.
type Location struct {
	Country    string
	State      string
	City       string
	PostalCode string
}

// Jurisdiction identifies a tax authority (country + optional subdivision).
type Jurisdiction struct {
	Country string
	State   string // province/region; empty means country-level
}

// Key returns a stable map key for the jurisdiction.
func (j Jurisdiction) Key() string {
	if j.State == "" {
		return j.Country
	}
	return j.Country + "/" + j.State
}

// TaxResult represents the calculated tax.
type TaxResult struct {
	TotalTax      commerce.Money
	Rate          float64
	Breakdown     map[string]commerce.Money // e.g. "state", "city", "country"
	TaxableAmount commerce.Money
	Jurisdiction  Jurisdiction
}

// Calculator defines the tax calculation interface.
type Calculator interface {
	// CalculateTax calculates tax for a given amount and location.
	CalculateTax(ctx context.Context, amount commerce.Money, loc Location) (*TaxResult, error)
}
