package workflow

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedWorkflowEngine wraps a WorkflowEngine with observability.
type InstrumentedWorkflowEngine struct {
	next   WorkflowEngine
	tracer trace.Tracer
}

// NewInstrumentedWorkflowEngine creates a new instrumented workflow engine.
func NewInstrumentedWorkflowEngine(next WorkflowEngine) *InstrumentedWorkflowEngine {
	return &InstrumentedWorkflowEngine{
		next:   next,
		tracer: otel.Tracer("pkg/workflow"),
	}
}

func (i *InstrumentedWorkflowEngine) RegisterWorkflow(ctx context.Context, def WorkflowDefinition) error {
	ctx, span := i.tracer.Start(ctx, "workflow.RegisterWorkflow", trace.WithAttributes(
		attribute.String("workflow.id", def.ID),
		attribute.String("workflow.name", def.Name),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "registering workflow", "id", def.ID, "name", def.Name)

	err := i.next.RegisterWorkflow(ctx, def)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedWorkflowEngine) GetWorkflow(ctx context.Context, workflowID string) (*WorkflowDefinition, error) {
	ctx, span := i.tracer.Start(ctx, "workflow.GetWorkflow", trace.WithAttributes(
		attribute.String("workflow.id", workflowID),
	))
	defer span.End()

	def, err := i.next.GetWorkflow(ctx, workflowID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return def, nil
}

func (i *InstrumentedWorkflowEngine) Start(ctx context.Context, opts StartOptions) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "workflow.Start", trace.WithAttributes(
		attribute.String("workflow.id", opts.WorkflowID),
		attribute.String("execution.id", opts.ExecutionID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "starting workflow execution",
		"workflow_id", opts.WorkflowID,
		"execution_id", opts.ExecutionID,
	)

	exec, err := i.next.Start(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "workflow start failed", "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("execution.id", exec.ID))
	return exec, nil
}

func (i *InstrumentedWorkflowEngine) GetExecution(ctx context.Context, executionID string) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "workflow.GetExecution", trace.WithAttributes(
		attribute.String("execution.id", executionID),
	))
	defer span.End()

	exec, err := i.next.GetExecution(ctx, executionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("execution.status", string(exec.Status)))
	return exec, nil
}

func (i *InstrumentedWorkflowEngine) ListExecutions(ctx context.Context, opts ListOptions) (*ListResult, error) {
	ctx, span := i.tracer.Start(ctx, "workflow.ListExecutions", trace.WithAttributes(
		attribute.String("workflow.id", opts.WorkflowID),
		attribute.String("execution.status", string(opts.Status)),
	))
	defer span.End()

	result, err := i.next.ListExecutions(ctx, opts)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("execution.count", len(result.Executions)))
	return result, nil
}

func (i *InstrumentedWorkflowEngine) Cancel(ctx context.Context, executionID string) error {
	ctx, span := i.tracer.Start(ctx, "workflow.Cancel", trace.WithAttributes(
		attribute.String("execution.id", executionID),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "cancelling workflow execution", "execution_id", executionID)

	err := i.next.Cancel(ctx, executionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedWorkflowEngine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	ctx, span := i.tracer.Start(ctx, "workflow.Signal", trace.WithAttributes(
		attribute.String("execution.id", executionID),
		attribute.String("signal.name", signalName),
	))
	defer span.End()

	err := i.next.Signal(ctx, executionID, signalName, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func (i *InstrumentedWorkflowEngine) Wait(ctx context.Context, executionID string) (*Execution, error) {
	ctx, span := i.tracer.Start(ctx, "workflow.Wait", trace.WithAttributes(
		attribute.String("execution.id", executionID),
	))
	defer span.End()

	exec, err := i.next.Wait(ctx, executionID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("execution.status", string(exec.Status)))
	return exec, nil
}

var _ WorkflowEngine = (*InstrumentedWorkflowEngine)(nil)
