package memory

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/tax"
)

// Calculator implements tax.Calculator in memory.
type Calculator struct {
	flatRate float64
}

// New creates a new memory tax calculator with a flat rate (default 10%).
func New() *Calculator {
	return &Calculator{flatRate: 0.10}
}

func (c *Calculator) CalculateTax(ctx context.Context, amount float64, loc tax.Location) (*tax.TaxResult, error) {
	// Simple logic: If country is "US", use flat rate. Else 0.
	rate := 0.0
	if loc.Country == "US" {
		rate = c.flatRate
	}

	total := amount * rate
	return &tax.TaxResult{
		TotalTax:      total,
		Rate:          rate,
		Breakdown:     map[string]float64{"flat": total},
		TaxableAmount: amount,
	}, nil
}
