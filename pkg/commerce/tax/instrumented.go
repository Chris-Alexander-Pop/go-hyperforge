package tax

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
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

func (c *InstrumentedCalculator) CalculateTax(ctx context.Context, amount float64, loc Location) (*TaxResult, error) {
	ctx, span := c.tracer.Start(ctx, "tax.CalculateTax", trace.WithAttributes(
		attribute.Float64("amount", amount),
		attribute.String("location.country", loc.Country),
		attribute.String("location.state", loc.State),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "calculating tax", "amount", amount, "country", loc.Country)

	res, err := c.next.CalculateTax(ctx, amount, loc)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to calculate tax", "error", err)
	} else {
		span.SetAttributes(attribute.Float64("tax.total", res.TotalTax))
	}
	return res, err
}
