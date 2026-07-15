package scheduler

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedScheduler wraps a Scheduler with logging and tracing.
type InstrumentedScheduler struct {
	next   Scheduler
	tracer trace.Tracer
}

// NewInstrumentedScheduler creates a new instrumented scheduler.
func NewInstrumentedScheduler(next Scheduler) *InstrumentedScheduler {
	return &InstrumentedScheduler{
		next:   next,
		tracer: otel.Tracer("pkg/cloud/scheduler"),
	}
}

func (s *InstrumentedScheduler) SelectHost(ctx context.Context, req Requirement) (string, error) {
	ctx, span := s.tracer.Start(ctx, "scheduler.SelectHost", trace.WithAttributes(
		attribute.Int("req.vcpus", req.Resources.VCPUs),
		attribute.Int("req.memory_mb", req.Resources.MemoryMB),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "selecting host", "vcpus", req.Resources.VCPUs, "memory", req.Resources.MemoryMB)

	hostID, err := s.next.SelectHost(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "failed to select host", "error", err)
		return "", err
	}

	span.SetAttributes(attribute.String("host.id", hostID))
	logger.L().InfoContext(ctx, "host selected", "id", hostID)
	return hostID, nil
}
