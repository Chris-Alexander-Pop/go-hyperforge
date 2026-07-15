package analytics

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedTracker wraps a Tracker with logging and tracing.
type InstrumentedTracker struct {
	next   Tracker
	tracer trace.Tracer
}

// NewInstrumentedTracker creates a new instrumented tracker.
func NewInstrumentedTracker(next Tracker) *InstrumentedTracker {
	return &InstrumentedTracker{
		next:   next,
		tracer: otel.Tracer("pkg/analytics"),
	}
}

// Ensure InstrumentedTracker implements Tracker.
var _ Tracker = (*InstrumentedTracker)(nil)

func (t *InstrumentedTracker) Add(ctx context.Context, counter string, element string) error {
	ctx, span := t.tracer.Start(ctx, "analytics.Add", trace.WithAttributes(
		attribute.String("counter.name", counter),
	))
	defer span.End()

	err := t.next.Add(ctx, counter, element)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to add to counter", "counter", counter, "error", err)
	}
	return err
}

func (t *InstrumentedTracker) Count(ctx context.Context, counter string) (uint64, error) {
	ctx, span := t.tracer.Start(ctx, "analytics.Count", trace.WithAttributes(
		attribute.String("counter.name", counter),
	))
	defer span.End()

	count, err := t.next.Count(ctx, counter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to count", "counter", counter, "error", err)
		return 0, err
	}

	span.SetAttributes(attribute.Int64("count", int64(count)))
	return count, nil
}

func (t *InstrumentedTracker) Reset(ctx context.Context, counter string) error {
	ctx, span := t.tracer.Start(ctx, "analytics.Reset", trace.WithAttributes(
		attribute.String("counter.name", counter),
	))
	defer span.End()

	err := t.next.Reset(ctx, counter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to reset counter", "counter", counter, "error", err)
	} else {
		logger.L().InfoContext(ctx, "counter reset", "counter", counter)
	}
	return err
}

func (t *InstrumentedTracker) Merge(ctx context.Context, dest, source string) error {
	ctx, span := t.tracer.Start(ctx, "analytics.Merge", trace.WithAttributes(
		attribute.String("counter.dest", dest),
		attribute.String("counter.source", source),
	))
	defer span.End()

	err := t.next.Merge(ctx, dest, source)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to merge counters",
			"dest", dest, "source", source, "error", err)
	}
	return err
}

func (t *InstrumentedTracker) Close() error {
	return t.next.Close()
}
