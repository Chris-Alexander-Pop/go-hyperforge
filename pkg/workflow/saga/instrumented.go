package saga

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedSaga wraps a Saga with logging and OpenTelemetry spans.
// Optional: use when you want observability without changing saga logic.
type InstrumentedSaga struct {
	next   *Saga
	tracer trace.Tracer
}

// NewInstrumentedSaga creates an instrumented saga wrapper.
func NewInstrumentedSaga(next *Saga) *InstrumentedSaga {
	return &InstrumentedSaga{
		next:   next,
		tracer: otel.Tracer("pkg/workflow/saga"),
	}
}

// Name returns the underlying saga name.
func (i *InstrumentedSaga) Name() string {
	return i.next.Name()
}

// AddStep adds a step to the underlying saga and returns the wrapper for chaining.
func (i *InstrumentedSaga) AddStep(step Step) *InstrumentedSaga {
	i.next.AddStep(step)
	return i
}

// Execute runs the saga with tracing and logging.
func (i *InstrumentedSaga) Execute(ctx context.Context, input interface{}) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "saga.Execute", trace.WithAttributes(
		attribute.String("saga.name", i.next.Name()),
		attribute.Int("saga.steps", len(i.next.Steps())),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "executing saga", "saga", i.next.Name())

	exec, err := i.next.Execute(ctx, input)
	if exec != nil {
		span.SetAttributes(
			attribute.String("saga.execution_id", exec.ID),
			attribute.String("saga.status", string(exec.Status)),
		)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "saga failed", "saga", i.next.Name(), "error", err)
		return exec, err
	}

	logger.L().InfoContext(ctx, "saga completed", "saga", i.next.Name(), "execution_id", exec.ID)
	return exec, nil
}
