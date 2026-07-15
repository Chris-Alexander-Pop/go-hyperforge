package tax

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedCalculator wraps a Calculator with logging and tracing.
type InstrumentedCalculator struct {
	next   Calculator
	tracer trace.Tracer
}

// NewInstrumentedCalculator creates a new InstrumentedCalculator.
func NewInstrumentedCalculator(next Calculator) *InstrumentedCalculator {
	return &InstrumentedCalculator{
		next:   next,
		tracer: otel.Tracer("pkg/commerce/tax"),
	}
}

func (c *InstrumentedCalculator) CalculateTax(ctx context.Context, amount commerce.Money, loc Location) (*TaxResult, error) {
	ctx, span := c.tracer.Start(ctx, "tax.CalculateTax", trace.WithAttributes(
		attribute.Int64("amount", amount.Amount),
		attribute.String("currency", amount.Currency),
		attribute.String("location.country", loc.Country),
		attribute.String("location.state", loc.State),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "calculating tax", "amount", amount.Amount, "country", loc.Country)

	res, err := c.next.CalculateTax(ctx, amount, loc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to calculate tax", "error", err)
	} else {
		span.SetAttributes(attribute.Int64("tax.total", res.TotalTax.Amount))
	}
	return res, err
}
