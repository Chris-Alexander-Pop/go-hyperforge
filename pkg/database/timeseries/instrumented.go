package timeseries

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedTimeseries wraps a Timeseries implementation with observability.
type InstrumentedTimeseries struct {
	next   Timeseries
	tracer trace.Tracer
}

// NewInstrumentedTimeseries creates a new instrumented wrapper.
func NewInstrumentedTimeseries(next Timeseries) *InstrumentedTimeseries {
	return &InstrumentedTimeseries{
		next:   next,
		tracer: otel.Tracer("pkg/database/timeseries"),
	}
}

// Write writes a single point with tracing and logging.
func (i *InstrumentedTimeseries) Write(ctx context.Context, point *Point) error {
	ctx, span := i.tracer.Start(ctx, "timeseries.Write", trace.WithAttributes(
		attribute.String("measurement", point.Measurement),
		attribute.Int("fields", len(point.Fields)),
	))
	defer span.End()

	err := i.next.Write(ctx, point)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "timeseries write failed", "error", err, "measurement", point.Measurement)
	}
	return err
}

// WriteBatch writes a batch of points with tracing and logging.
func (i *InstrumentedTimeseries) WriteBatch(ctx context.Context, points []*Point) error {
	ctx, span := i.tracer.Start(ctx, "timeseries.WriteBatch", trace.WithAttributes(
		attribute.Int("batch_size", len(points)),
	))
	defer span.End()

	err := i.next.WriteBatch(ctx, points)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "timeseries batch write failed", "error", err, "batch_size", len(points))
	}
	return err
}

// Query executes a query with tracing and logging.
func (i *InstrumentedTimeseries) Query(ctx context.Context, query string) ([]*Point, error) {
	ctx, span := i.tracer.Start(ctx, "timeseries.Query", trace.WithAttributes(
		attribute.String("query", query),
	))
	defer span.End()

	results, err := i.next.Query(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "timeseries query failed", "error", err, "query", query)
	} else {
		span.SetAttributes(attribute.Int("result_count", len(results)))
	}
	return results, err
}

// Close closes the underlying connection.
func (i *InstrumentedTimeseries) Close() error {
	return i.next.Close()
}
