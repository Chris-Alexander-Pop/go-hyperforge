package workflow

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	"github.com/google/uuid"
)

// Ensure EventedEngine implements WorkflowEngine at compile time.
var _ WorkflowEngine = (*EventedEngine)(nil)

const (
	// TopicWorkflow is the pkg/events topic for workflow domain events.
	TopicWorkflow = "workflow"

	// EventTypeStarted is emitted after a successful Start.
	EventTypeStarted = "workflow.execution.started"

	// EventTypeCompleted is emitted when Wait observes a completed execution.
	EventTypeCompleted = "workflow.execution.completed"

	// EventTypeFailed is emitted when Wait observes a failed or timed-out execution,
	// or when Start fails after the engine returns an error that indicates failure
	// of an in-flight attempt (Start API errors do not emit).
	EventTypeFailed = "workflow.execution.failed"
)

// ExecutionEventPayload is the typed payload for workflow execution lifecycle events.
type ExecutionEventPayload struct {
	ExecutionID string          `json:"execution_id"`
	WorkflowID  string          `json:"workflow_id"`
	Status      ExecutionStatus `json:"status"`
	Error       string          `json:"error,omitempty"`
}

// EventedEngine decorates a WorkflowEngine to publish lifecycle events via pkg/events.
// Publish is best-effort: failures are ignored so orchestration is not rolled back.
//
// Events:
//   - workflow.execution.started after successful Start
//   - workflow.execution.completed / failed after Wait returns a terminal status
type EventedEngine struct {
	next WorkflowEngine
	bus  events.Bus
}

// NewEventedEngine wraps next so Start/Wait fan out to bus.
// If bus is nil, publishing is skipped and operations still delegate to next.
func NewEventedEngine(next WorkflowEngine, bus events.Bus) *EventedEngine {
	return &EventedEngine{next: next, bus: bus}
}

func (e *EventedEngine) publish(ctx context.Context, eventType string, exec *Execution) {
	if e.bus == nil || exec == nil {
		return
	}
	id := exec.ID
	if id == "" {
		id = uuid.NewString()
	}
	_ = e.bus.Publish(ctx, TopicWorkflow, events.Event{
		ID:        id + ":" + eventType,
		Type:      eventType,
		Source:    "pkg/workflow",
		Timestamp: time.Now().UTC(),
		Payload: ExecutionEventPayload{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Status:      exec.Status,
			Error:       exec.Error,
		},
	})
}

// RegisterWorkflow delegates to the underlying engine.
func (e *EventedEngine) RegisterWorkflow(ctx context.Context, def WorkflowDefinition) error {
	return e.next.RegisterWorkflow(ctx, def)
}

// GetWorkflow delegates to the underlying engine.
func (e *EventedEngine) GetWorkflow(ctx context.Context, workflowID string) (*WorkflowDefinition, error) {
	return e.next.GetWorkflow(ctx, workflowID)
}

// Start starts an execution and emits workflow.execution.started on success.
func (e *EventedEngine) Start(ctx context.Context, opts StartOptions) (*Execution, error) {
	exec, err := e.next.Start(ctx, opts)
	if err != nil {
		return nil, err
	}
	e.publish(ctx, EventTypeStarted, exec)
	return exec, nil
}

// GetExecution delegates to the underlying engine.
func (e *EventedEngine) GetExecution(ctx context.Context, executionID string) (*Execution, error) {
	return e.next.GetExecution(ctx, executionID)
}

// ListExecutions delegates to the underlying engine.
func (e *EventedEngine) ListExecutions(ctx context.Context, opts ListOptions) (*ListResult, error) {
	return e.next.ListExecutions(ctx, opts)
}

// Cancel delegates to the underlying engine.
func (e *EventedEngine) Cancel(ctx context.Context, executionID string) error {
	return e.next.Cancel(ctx, executionID)
}

// Signal delegates to the underlying engine.
func (e *EventedEngine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	return e.next.Signal(ctx, executionID, signalName, data)
}

// Wait waits for completion and emits completed or failed based on terminal status.
func (e *EventedEngine) Wait(ctx context.Context, executionID string) (*Execution, error) {
	exec, err := e.next.Wait(ctx, executionID)
	if err != nil {
		return nil, err
	}
	switch exec.Status {
	case StatusCompleted:
		e.publish(ctx, EventTypeCompleted, exec)
	case StatusFailed, StatusTimedOut:
		e.publish(ctx, EventTypeFailed, exec)
	}
	return exec, nil
}
