package saga

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedDurableExecutor wraps DurableExecutor with logging and tracing.
type InstrumentedDurableExecutor struct {
	next   *DurableExecutor
	tracer trace.Tracer
}

// NewInstrumentedDurableExecutor creates an instrumented durable saga executor.
func NewInstrumentedDurableExecutor(next *DurableExecutor) *InstrumentedDurableExecutor {
	return &InstrumentedDurableExecutor{
		next:   next,
		tracer: otel.Tracer("pkg/workflow/saga"),
	}
}

// Execute starts a durable saga with observability.
func (i *InstrumentedDurableExecutor) Execute(ctx context.Context, sagaName string, input any) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "saga.DurableExecute", trace.WithAttributes(
		attribute.String("saga.name", sagaName),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "executing durable saga", "saga", sagaName)

	exec, err := i.next.Execute(ctx, sagaName, input)
	if exec != nil {
		span.SetAttributes(
			attribute.String("saga.execution_id", exec.ID),
			attribute.String("saga.status", string(exec.Status)),
		)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "durable saga failed", "saga", sagaName, "error", err)
		return exec, err
	}
	logger.L().InfoContext(ctx, "durable saga completed", "saga", sagaName, "execution_id", exec.ID)
	return exec, nil
}

// Resume continues a persisted execution with observability.
func (i *InstrumentedDurableExecutor) Resume(ctx context.Context, executionID string) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "saga.DurableResume", trace.WithAttributes(
		attribute.String("saga.execution_id", executionID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "resuming durable saga", "execution_id", executionID)

	exec, err := i.next.Resume(ctx, executionID)
	if exec != nil {
		span.SetAttributes(
			attribute.String("saga.name", exec.SagaName),
			attribute.String("saga.status", string(exec.Status)),
		)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "durable saga resume failed", "execution_id", executionID, "error", err)
		return exec, err
	}
	return exec, nil
}

// ResumeAll resumes incomplete executions with observability.
func (i *InstrumentedDurableExecutor) ResumeAll(ctx context.Context) ([]*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "saga.DurableResumeAll")
	defer span.End()

	out, err := i.next.ResumeAll(ctx)
	span.SetAttributes(attribute.Int("saga.resume_count", len(out)))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return out, err
	}
	logger.L().InfoContext(ctx, "resumed durable sagas", "count", len(out))
	return out, nil
}
