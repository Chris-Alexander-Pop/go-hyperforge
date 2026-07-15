package memory

import (
	"context"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/tax"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
)

// Ensure compile-time interface compliance.
var _ tax.Calculator = (*Calculator)(nil)

// rateEntry holds jurisdiction tax rates (fraction, e.g. 0.08 = 8%).
type rateEntry struct {
	country float64
	state   float64
	city    float64
}

// Calculator implements tax.Calculator with a multi-jurisdiction rate table.
type Calculator struct {
	rates map[string]rateEntry // key: Country or Country/State
	mu    *concurrency.SmartRWMutex
}

// New creates a memory tax calculator with sample US state rates.
func New() *Calculator {
	return &Calculator{
		rates: map[string]rateEntry{
			"US":    {country: 0, state: 0.05, city: 0},
			"US/NY": {country: 0, state: 0.04, city: 0.045},
			"US/CA": {country: 0, state: 0.0725, city: 0},
			"US/TX": {country: 0, state: 0.0625, city: 0},
			"CA":    {country: 0.05, state: 0, city: 0},
			"CA/ON": {country: 0.05, state: 0.08, city: 0},
			"GB":    {country: 0.20, state: 0, city: 0},
			"DE":    {country: 0.19, state: 0, city: 0},
		},
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "commerce-tax-memory"}),
	}
}

// SetRate configures or replaces a jurisdiction rate (test helper).
func (c *Calculator) SetRate(j tax.Jurisdiction, country, state, city float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rates[normalizeKey(j.Country, j.State)] = rateEntry{country: country, state: state, city: city}
}

func (c *Calculator) CalculateTax(ctx context.Context, amount commerce.Money, loc tax.Location) (*tax.TaxResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := amount.Validate(); err != nil {
		return nil, tax.ErrInvalidAmount
	}
	if amount.Amount < 0 {
		return nil, tax.ErrInvalidAmount
	}
	if strings.TrimSpace(loc.Country) == "" {
		return nil, tax.ErrUnsupportedLocation
	}

	j := tax.Jurisdiction{
		Country: strings.ToUpper(loc.Country),
		State:   strings.ToUpper(loc.State),
	}

	c.mu.RLock()
	entry, ok := c.lookupLocked(j)
	c.mu.RUnlock()
	if !ok {
		// Unknown jurisdiction: zero tax rather than hard-fail (marketplace default).
		return &tax.TaxResult{
			TotalTax:      commerce.Zero(amount.Currency),
			Rate:          0,
			Breakdown:     map[string]commerce.Money{},
			TaxableAmount: amount,
			Jurisdiction:  j,
		}, nil
	}

	rate := entry.country + entry.state + entry.city
	totalMinor := roundMinor(float64(amount.Amount) * rate)
	breakdown := map[string]commerce.Money{}
	if entry.country > 0 {
		breakdown["country"] = commerce.NewMoney(roundMinor(float64(amount.Amount)*entry.country), amount.Currency)
	}
	if entry.state > 0 {
		breakdown["state"] = commerce.NewMoney(roundMinor(float64(amount.Amount)*entry.state), amount.Currency)
	}
	if entry.city > 0 {
		breakdown["city"] = commerce.NewMoney(roundMinor(float64(amount.Amount)*entry.city), amount.Currency)
	}

	return &tax.TaxResult{
		TotalTax:      commerce.NewMoney(totalMinor, amount.Currency),
		Rate:          rate,
		Breakdown:     breakdown,
		TaxableAmount: amount,
		Jurisdiction:  j,
	}, nil
}

func (c *Calculator) lookupLocked(j tax.Jurisdiction) (rateEntry, bool) {
	if j.State != "" {
		if e, ok := c.rates[normalizeKey(j.Country, j.State)]; ok {
			return e, true
		}
	}
	if e, ok := c.rates[normalizeKey(j.Country, "")]; ok {
		return e, true
	}
	return rateEntry{}, false
}

func normalizeKey(country, state string) string {
	country = strings.ToUpper(strings.TrimSpace(country))
	state = strings.ToUpper(strings.TrimSpace(state))
	if state == "" {
		return country
	}
	return country + "/" + state
}

func roundMinor(v float64) int64 {
	if v < 0 {
		return int64(v - 0.5)
	}
	return int64(v + 0.5)
}
