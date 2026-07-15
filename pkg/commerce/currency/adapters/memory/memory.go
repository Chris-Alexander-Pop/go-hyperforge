package memory

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/currency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
)

// Ensure compile-time interface compliance.
var _ currency.Converter = (*Converter)(nil)

// Converter implements currency.Converter in memory with a static rate table.
// It does not implement LiveRateProvider — rates stay static unless replaced
// via SetRates (test/helper).
type Converter struct {
	rates map[string]float64 // Rates relative to USD base
	mu    *concurrency.SmartRWMutex
}

// New creates a new memory currency converter.
func New() *Converter {
	return &Converter{
		rates: map[string]float64{
			"USD": 1.0,
			"EUR": 0.85,
			"GBP": 0.75,
			"JPY": 110.0,
		},
		mu: concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "commerce-currency-memory"}),
	}
}

// SetRates replaces the static rate table (test helper; not a live feed).
func (c *Converter) SetRates(rates map[string]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rates = make(map[string]float64, len(rates))
	for k, v := range rates {
		c.rates[strings.ToUpper(k)] = v
	}
}

func (c *Converter) GetRate(ctx context.Context, from string, to string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	c.mu.RLock()
	defer c.mu.RUnlock()

	r1, ok1 := c.rates[from]
	r2, ok2 := c.rates[to]
	if !ok1 || !ok2 {
		return 0, currency.ErrUnsupportedCurrency
	}
	return r2 / r1, nil
}

func (c *Converter) Convert(ctx context.Context, amount float64, from string, to string) (*currency.ConversionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return nil, currency.ErrInvalidAmount
	}

	rate, err := c.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	return &currency.ConversionResult{
		FromAmount: amount,
		From:       strings.ToUpper(from),
		ToAmount:   amount * rate,
		To:         strings.ToUpper(to),
		Rate:       rate,
		Timestamp:  time.Now().UTC(),
	}, nil
}
