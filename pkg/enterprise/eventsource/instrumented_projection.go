package eventsource

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProjectionRunner wraps ProjectionRunner with logging and tracing.
type InstrumentedProjectionRunner struct {
	next   *ProjectionRunner
	tracer trace.Tracer
}

// NewInstrumentedProjectionRunner decorates a ProjectionRunner.
func NewInstrumentedProjectionRunner(next *ProjectionRunner) *InstrumentedProjectionRunner {
	return &InstrumentedProjectionRunner{
		next:   next,
		tracer: otel.Tracer("pkg/enterprise/eventsource"),
	}
}

// Name delegates to the underlying runner.
func (r *InstrumentedProjectionRunner) Name() string { return r.next.Name() }

// Checkpoint delegates to the underlying runner.
func (r *InstrumentedProjectionRunner) Checkpoint(ctx context.Context) (Checkpoint, error) {
	return r.next.Checkpoint(ctx)
}

// ResetCheckpoint logs and delegates.
func (r *InstrumentedProjectionRunner) ResetCheckpoint(ctx context.Context) error {
	ctx, span := r.tracer.Start(ctx, "eventsource.Projection.ResetCheckpoint", trace.WithAttributes(
		attribute.String("projection.name", r.next.Name()),
	))
	defer span.End()
	err := r.next.ResetCheckpoint(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "projection reset checkpoint failed", "name", r.next.Name(), "error", err)
		return err
	}
	logger.L().InfoContext(ctx, "projection checkpoint reset", "name", r.next.Name())
	return nil
}

// RunOnce logs/traces a single catch-up pass.
func (r *InstrumentedProjectionRunner) RunOnce(ctx context.Context) error {
	ctx, span := r.tracer.Start(ctx, "eventsource.Projection.RunOnce", trace.WithAttributes(
		attribute.String("projection.name", r.next.Name()),
	))
	defer span.End()

	start := time.Now()
	err := r.next.RunOnce(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "projection run_once failed",
			"name", r.next.Name(), "error", err, "duration_ms", time.Since(start).Milliseconds())
		return err
	}
	logger.L().DebugContext(ctx, "projection run_once ok",
		"name", r.next.Name(), "duration_ms", time.Since(start).Milliseconds())
	return nil
}

// Run continuously projects with instrumentation on each cycle's errors via next.
func (r *InstrumentedProjectionRunner) Run(ctx context.Context) error {
	ctx, span := r.tracer.Start(ctx, "eventsource.Projection.Run", trace.WithAttributes(
		attribute.String("projection.name", r.next.Name()),
	))
	defer span.End()
	logger.L().InfoContext(ctx, "projection run started", "name", r.next.Name())
	err := r.next.Run(ctx)
	if err != nil && ctx.Err() == nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "projection run stopped with error", "name", r.next.Name(), "error", err)
		return err
	}
	logger.L().InfoContext(ctx, "projection run stopped", "name", r.next.Name())
	return err
}
