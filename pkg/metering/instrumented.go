package metering

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure instrumented wrappers satisfy their interfaces at compile time.
var (
	_ Meter = (*InstrumentedMeter)(nil)
	_ Rater = (*InstrumentedRater)(nil)
)

// InstrumentedMeter wraps a Meter with logging and tracing.
type InstrumentedMeter struct {
	next   Meter
	tracer trace.Tracer
}

// NewInstrumentedMeter creates a new instrumented meter.
func NewInstrumentedMeter(next Meter) *InstrumentedMeter {
	return &InstrumentedMeter{
		next:   next,
		tracer: otel.Tracer("pkg/metering"),
	}
}

func (m *InstrumentedMeter) RecordUsage(ctx context.Context, event UsageEvent) error {
	ctx, span := m.tracer.Start(ctx, "metering.RecordUsage", trace.WithAttributes(
		attribute.String("tenant.id", event.TenantID),
		attribute.String("resource.type", event.ResourceType),
		attribute.Float64("quantity", event.Quantity),
	))
	defer span.End()

	start := time.Now()
	err := m.next.RecordUsage(ctx, event)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to record usage",
			"operation", "metering.RecordUsage",
			"tenant_id", event.TenantID,
			"resource_type", event.ResourceType,
			"quantity", event.Quantity,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}

	logger.L().InfoContext(ctx, "recording usage",
		"operation", "metering.RecordUsage",
		"tenant_id", event.TenantID,
		"resource_type", event.ResourceType,
		"resource_id", event.ResourceID,
		"quantity", event.Quantity,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

func (m *InstrumentedMeter) GetUsage(ctx context.Context, filter UsageFilter) ([]UsageEvent, error) {
	ctx, span := m.tracer.Start(ctx, "metering.GetUsage", trace.WithAttributes(
		attribute.String("tenant.id", filter.TenantID),
		attribute.String("resource.type", filter.ResourceType),
	))
	defer span.End()

	start := time.Now()
	events, err := m.next.GetUsage(ctx, filter)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get usage",
			"operation", "metering.GetUsage",
			"tenant_id", filter.TenantID,
			"resource_type", filter.ResourceType,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("result.count", len(events)))
	logger.L().DebugContext(ctx, "got usage",
		"operation", "metering.GetUsage",
		"tenant_id", filter.TenantID,
		"resource_type", filter.ResourceType,
		"count", len(events),
		"duration_ms", duration.Milliseconds(),
	)
	return events, nil
}

func (m *InstrumentedMeter) Close() error {
	return m.next.Close()
}

// InstrumentedRater wraps a Rater with logging and tracing.
type InstrumentedRater struct {
	next   Rater
	tracer trace.Tracer
}

// NewInstrumentedRater creates a new instrumented rater.
func NewInstrumentedRater(next Rater) *InstrumentedRater {
	return &InstrumentedRater{
		next:   next,
		tracer: otel.Tracer("pkg/metering"),
	}
}

func (r *InstrumentedRater) GetRate(ctx context.Context, resourceType string) (*RateCard, error) {
	ctx, span := r.tracer.Start(ctx, "metering.GetRate", trace.WithAttributes(
		attribute.String("resource.type", resourceType),
	))
	defer span.End()

	start := time.Now()
	rate, err := r.next.GetRate(ctx, resourceType)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to get rate",
			"operation", "metering.GetRate",
			"resource_type", resourceType,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	logger.L().DebugContext(ctx, "got rate",
		"operation", "metering.GetRate",
		"resource_type", resourceType,
		"price_per_unit", rate.PricePerUnit,
		"currency", rate.Currency,
		"duration_ms", duration.Milliseconds(),
	)
	return rate, nil
}

func (r *InstrumentedRater) SetRate(ctx context.Context, rate RateCard) error {
	ctx, span := r.tracer.Start(ctx, "metering.SetRate", trace.WithAttributes(
		attribute.String("resource.type", rate.ResourceType),
		attribute.Float64("price_per_unit", rate.PricePerUnit),
		attribute.String("currency", rate.Currency),
	))
	defer span.End()

	start := time.Now()
	err := r.next.SetRate(ctx, rate)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to set rate",
			"operation", "metering.SetRate",
			"resource_type", rate.ResourceType,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return err
	}

	logger.L().InfoContext(ctx, "set rate",
		"operation", "metering.SetRate",
		"resource_type", rate.ResourceType,
		"price_per_unit", rate.PricePerUnit,
		"currency", rate.Currency,
		"unit", rate.Unit,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

func (r *InstrumentedRater) ListRates(ctx context.Context) ([]RateCard, error) {
	ctx, span := r.tracer.Start(ctx, "metering.ListRates")
	defer span.End()

	start := time.Now()
	rates, err := r.next.ListRates(ctx)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to list rates",
			"operation", "metering.ListRates",
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Int("result.count", len(rates)))
	logger.L().DebugContext(ctx, "listed rates",
		"operation", "metering.ListRates",
		"count", len(rates),
		"duration_ms", duration.Milliseconds(),
	)
	return rates, nil
}

func (r *InstrumentedRater) CalculateCost(ctx context.Context, usage UsageEvent) (float64, error) {
	ctx, span := r.tracer.Start(ctx, "metering.CalculateCost", trace.WithAttributes(
		attribute.String("tenant.id", usage.TenantID),
		attribute.String("resource.type", usage.ResourceType),
		attribute.Float64("quantity", usage.Quantity),
	))
	defer span.End()

	start := time.Now()
	cost, err := r.next.CalculateCost(ctx, usage)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to calculate cost",
			"operation", "metering.CalculateCost",
			"tenant_id", usage.TenantID,
			"resource_type", usage.ResourceType,
			"quantity", usage.Quantity,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return 0, err
	}

	span.SetAttributes(attribute.Float64("cost", cost))
	logger.L().InfoContext(ctx, "calculated cost",
		"operation", "metering.CalculateCost",
		"tenant_id", usage.TenantID,
		"resource_type", usage.ResourceType,
		"quantity", usage.Quantity,
		"cost", cost,
		"duration_ms", duration.Milliseconds(),
	)
	return cost, nil
}

func (r *InstrumentedRater) Close() error {
	return r.next.Close()
}
