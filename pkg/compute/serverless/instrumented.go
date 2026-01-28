package serverless

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedServerlessRuntime wraps a ServerlessRuntime with observability.
type InstrumentedServerlessRuntime struct {
	next   ServerlessRuntime
	tracer trace.Tracer
}

// NewInstrumentedServerlessRuntime creates a new instrumented serverless runtime.
func NewInstrumentedServerlessRuntime(next ServerlessRuntime) *InstrumentedServerlessRuntime {
	return &InstrumentedServerlessRuntime{
		next:   next,
		tracer: otel.Tracer("pkg/compute/serverless"),
	}
}

func (i *InstrumentedServerlessRuntime) CreateFunction(ctx context.Context, opts CreateFunctionOptions) (*Function, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.CreateFunction", trace.WithAttributes(
		attribute.String("function.name", opts.Name),
		attribute.String("function.runtime", string(opts.Runtime)),
		attribute.String("function.handler", opts.Handler),
		attribute.Int("function.memory_mb", opts.MemoryMB),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "creating function", "name", opts.Name, "runtime", opts.Runtime)

	fn, err := i.next.CreateFunction(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "function creation failed", "name", opts.Name, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("function.arn", fn.ARN))
	return fn, nil
}

func (i *InstrumentedServerlessRuntime) GetFunction(ctx context.Context, name string) (*Function, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.GetFunction", trace.WithAttributes(
		attribute.String("function.name", name),
	))
	defer span.End()

	fn, err := i.next.GetFunction(ctx, name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return fn, nil
}

func (i *InstrumentedServerlessRuntime) ListFunctions(ctx context.Context) ([]*Function, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.ListFunctions")
	defer span.End()

	functions, err := i.next.ListFunctions(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("function.count", len(functions)))
	return functions, nil
}

func (i *InstrumentedServerlessRuntime) UpdateFunction(ctx context.Context, name string, opts CreateFunctionOptions) (*Function, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.UpdateFunction", trace.WithAttributes(
		attribute.String("function.name", name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "updating function", "name", name)

	fn, err := i.next.UpdateFunction(ctx, name, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "function update failed", "name", name, "error", err)
		return nil, err
	}
	return fn, nil
}

func (i *InstrumentedServerlessRuntime) DeleteFunction(ctx context.Context, name string) error {
	ctx, span := i.tracer.Start(ctx, "serverless.DeleteFunction", trace.WithAttributes(
		attribute.String("function.name", name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "deleting function", "name", name)

	err := i.next.DeleteFunction(ctx, name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "function deletion failed", "name", name, "error", err)
		return err
	}
	return nil
}

func (i *InstrumentedServerlessRuntime) Invoke(ctx context.Context, opts InvokeOptions) (*InvokeResult, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.Invoke", trace.WithAttributes(
		attribute.String("function.name", opts.FunctionName),
		attribute.String("function.invocation_type", string(opts.InvocationType)),
		attribute.Int("function.payload_size", len(opts.Payload)),
	))
	defer span.End()

	result, err := i.next.Invoke(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "function invocation failed", "name", opts.FunctionName, "error", err)
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("function.status_code", result.StatusCode),
		attribute.Int("function.response_size", len(result.Payload)),
	)
	return result, nil
}

func (i *InstrumentedServerlessRuntime) InvokeSimple(ctx context.Context, name string, payload []byte) ([]byte, error) {
	ctx, span := i.tracer.Start(ctx, "serverless.InvokeSimple", trace.WithAttributes(
		attribute.String("function.name", name),
		attribute.Int("function.payload_size", len(payload)),
	))
	defer span.End()

	result, err := i.next.InvokeSimple(ctx, name, payload)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("function.response_size", len(result)))
	return result, nil
}

var _ ServerlessRuntime = (*InstrumentedServerlessRuntime)(nil)
