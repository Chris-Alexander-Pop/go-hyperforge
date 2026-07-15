// Package temporal provides a Temporal.io adapter for workflow.WorkflowEngine.
//
// Temporal provides durable execution for long-running workflows with automatic
// retries, timeouts, and visibility.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/temporal"
//
//	engine, err := temporal.New(temporal.Config{Host: "localhost:7233", Namespace: "default"})
//	exec, err := engine.Start(ctx, workflow.StartOptions{WorkflowID: "order-123", Input: data})
package temporal

import (
	"context"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"go.temporal.io/sdk/client"
)

// Config holds Temporal configuration.
type Config struct {
	// Host is the Temporal server address.
	Host string

	// Namespace is the Temporal namespace.
	Namespace string

	// TaskQueue is the default task queue.
	TaskQueue string
}

// Engine implements workflow.WorkflowEngine for Temporal.
type Engine struct {
	client    client.Client
	config    Config
	workflows map[string]interface{} // workflow type registry
}

// New creates a new Temporal engine.
func New(cfg Config) (*Engine, error) {
	if cfg.Host == "" {
		cfg.Host = "localhost:7233"
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = "default-task-queue"
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.Host,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to connect to Temporal", err)
	}

	return &Engine{
		client:    c,
		config:    cfg,
		workflows: make(map[string]interface{}),
	}, nil
}

// Close closes the Temporal client.
func (e *Engine) Close() {
	if e.client != nil {
		e.client.Close()
	}
}

// RegisterWorkflowType registers a workflow function type for execution.
func (e *Engine) RegisterWorkflowType(name string, workflowFunc interface{}) {
	e.workflows[name] = workflowFunc
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	// Temporal workflows are registered via worker, not the engine
	e.workflows[def.ID] = def
	return nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	def, ok := e.workflows[workflowID]
	if !ok {
		return nil, pkgerrors.NotFound("workflow not registered", nil)
	}

	if wfDef, ok := def.(workflow.WorkflowDefinition); ok {
		return &wfDef, nil
	}

	return &workflow.WorkflowDefinition{
		ID:   workflowID,
		Name: workflowID,
	}, nil
}

func (e *Engine) Start(ctx context.Context, opts workflow.StartOptions) (*workflow.Execution, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        opts.ExecutionID,
		TaskQueue: e.config.TaskQueue,
	}

	if opts.ExecutionID == "" {
		workflowOptions.ID = opts.WorkflowID
	}

	if opts.Timeout > 0 {
		workflowOptions.WorkflowExecutionTimeout = opts.Timeout
	}

	run, err := e.client.ExecuteWorkflow(ctx, workflowOptions, opts.WorkflowID, opts.Input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to start workflow", err)
	}

	return &workflow.Execution{
		ID:         run.GetRunID(),
		WorkflowID: run.GetID(),
		Status:     workflow.StatusRunning,
		Input:      opts.Input,
		StartedAt:  time.Now(),
	}, nil
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	resp, err := e.client.DescribeWorkflowExecution(ctx, executionID, "")
	if err != nil {
		return nil, pkgerrors.NotFound("execution not found", err)
	}

	info := resp.WorkflowExecutionInfo
	exec := &workflow.Execution{
		ID:         info.Execution.RunId,
		WorkflowID: info.Execution.WorkflowId,
		Status:     mapTemporalStatus(info.Status),
	}

	if info.StartTime != nil {
		exec.StartedAt = info.StartTime.AsTime()
	}
	if info.CloseTime != nil {
		exec.CompletedAt = info.CloseTime.AsTime()
	}

	return exec, nil
}

func mapTemporalStatus(status interface{}) workflow.ExecutionStatus {
	switch status {
	case 1: // Running
		return workflow.StatusRunning
	case 2: // Completed
		return workflow.StatusCompleted
	case 3: // Failed
		return workflow.StatusFailed
	case 4: // Canceled
		return workflow.StatusCancelled
	case 5: // Terminated
		return workflow.StatusCancelled
	case 6: // ContinuedAsNew
		return workflow.StatusRunning
	case 7: // TimedOut
		return workflow.StatusTimedOut
	default:
		return workflow.StatusPending
	}
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	// Temporal list is complex - for now return empty for this SDK wrapper
	// Real usage would use visibility queries
	return &workflow.ListResult{
		Executions: make([]*workflow.Execution, 0),
	}, nil
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	err := e.client.CancelWorkflow(ctx, executionID, "")
	if err != nil {
		return pkgerrors.Internal("failed to cancel workflow", err)
	}
	return nil
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	err := e.client.SignalWorkflow(ctx, executionID, "", signalName, data)
	if err != nil {
		return pkgerrors.Internal("failed to signal workflow", err)
	}
	return nil
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	run := e.client.GetWorkflow(ctx, executionID, "")

	var result interface{}
	err := run.Get(ctx, &result)

	exec := &workflow.Execution{
		ID:          run.GetRunID(),
		WorkflowID:  run.GetID(),
		CompletedAt: time.Now(),
	}

	if err != nil {
		exec.Status = workflow.StatusFailed
		exec.Error = err.Error()
	} else {
		exec.Status = workflow.StatusCompleted
		exec.Output = result
	}

	return exec, nil
}
