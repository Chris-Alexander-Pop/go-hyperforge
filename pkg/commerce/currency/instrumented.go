package currency

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedConverter wraps a Converter with logging and tracing.
type InstrumentedConverter struct {
	next   Converter
	tracer trace.Tracer
}

// NewInstrumentedConverter creates a new InstrumentedConverter.
func NewInstrumentedConverter(next Converter) *InstrumentedConverter {
	return &InstrumentedConverter{
		next:   next,
		tracer: otel.Tracer("pkg/commerce/currency"),
	}
}

func (c *InstrumentedConverter) Convert(ctx context.Context, amount float64, from string, to string) (*ConversionResult, error) {
	ctx, span := c.tracer.Start(ctx, "currency.Convert", trace.WithAttributes(
		attribute.Float64("amount", amount),
		attribute.String("from", from),
		attribute.String("to", to),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "converting currency", "amount", amount, "from", from, "to", to)

	res, err := c.next.Convert(ctx, amount, from, to)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to convert currency", "error", err)
	} else {
		span.SetAttributes(attribute.Float64("result.amount", res.ToAmount))
	}
	return res, err
}

func (c *InstrumentedConverter) GetRate(ctx context.Context, from string, to string) (float64, error) {
	ctx, span := c.tracer.Start(ctx, "currency.GetRate", trace.WithAttributes(
		attribute.String("from", from),
		attribute.String("to", to),
	))
	defer span.End()

	rate, err := c.next.GetRate(ctx, from, to)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return rate, err
}
