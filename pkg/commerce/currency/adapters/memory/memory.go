package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/currency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Converter implements currency.Converter in memory.
type Converter struct {
	rates map[string]float64 // Rates relative to USD base
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
	}
}

func (c *Converter) GetRate(ctx context.Context, from string, to string) (float64, error) {
	r1, ok1 := c.rates[from]
	r2, ok2 := c.rates[to]

	if !ok1 || !ok2 {
		return 0, errors.NotFound("currency not supported", nil)
	}

	// Convert from -> USD -> to
	// Amount in USD = AmountInFrom / r1
	// Amount in To = AmountInUSD * r2 = (AmountInFrom / r1) * r2
	// Rate = r2 / r1
	return r2 / r1, nil
}

func (c *Converter) Convert(ctx context.Context, amount float64, from string, to string) (*currency.ConversionResult, error) {
	rate, err := c.GetRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	converted := amount * rate
	return &currency.ConversionResult{
		FromAmount: amount,
		From:       from,
		ToAmount:   converted,
		To:         to,
		Rate:       rate,
		Timestamp:  time.Now(),
	}, nil
}
