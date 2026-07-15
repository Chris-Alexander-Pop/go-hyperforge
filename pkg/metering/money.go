package metering

import (
	"context"
	"math"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// CalculateCostMoney rates usage into commerce.Money (minor units).
// PricePerUnit on the rate card is interpreted as major currency units per usage unit
// (e.g. 0.02 USD/hour → 2 cents per hour when quantity=1).
func CalculateCostMoney(ctx context.Context, rater Rater, usage UsageEvent) (commerce.Money, error) {
	if rater == nil {
		return commerce.Money{}, errors.InvalidArgument("rater is required", nil)
	}
	if err := ValidateUsageEvent(usage); err != nil {
		return commerce.Money{}, err
	}
	rate, err := rater.GetRate(ctx, usage.ResourceType)
	if err != nil {
		return commerce.Money{}, err
	}
	major := usage.Quantity * rate.PricePerUnit
	dec := commerce.Decimals(rate.Currency)
	scale := math.Pow10(dec)
	minor := int64(math.Round(major * scale))
	return commerce.NewMoney(minor, rate.Currency), nil
}
