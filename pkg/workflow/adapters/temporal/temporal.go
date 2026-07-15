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
//	defer engine.Close()
package temporal

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const defaultTaskQueue = "default-task-queue"

// Config holds Temporal configuration.
type Config struct {
	// Host is the Temporal server address.
	Host string

	// Namespace is the Temporal namespace.
	Namespace string

	// TaskQueue is the default task queue.
	TaskQueue string
}

// ClientAPI is the Temporal client surface used by this adapter (for tests).
type ClientAPI interface {
	ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error)
	DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error)
	ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error)
	CancelWorkflow(ctx context.Context, workflowID, runID string) error
	SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error
	GetWorkflow(ctx context.Context, workflowID, runID string) client.WorkflowRun
	Close()
}

// Engine implements workflow.WorkflowEngine for Temporal.
type Engine struct {
	client     ClientAPI
	config     Config
	workflows  map[string]interface{} // workflow type registry
	ownsClient bool
}

// New creates a new Temporal engine connected to a live Temporal server.
func New(cfg Config) (*Engine, error) {
	if cfg.Host == "" {
		cfg.Host = "localhost:7233"
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = defaultTaskQueue
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.Host,
		Namespace: cfg.Namespace,
	})
	if err != nil {
		return nil, pkgerrors.Internal("failed to connect to Temporal", err)
	}

	return NewFromClient(c, cfg, true), nil
}

// NewFromClient wraps an existing Temporal client (or test double).
// When ownsClient is true, Close() closes the underlying client.
func NewFromClient(c ClientAPI, cfg Config, ownsClient bool) *Engine {
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.TaskQueue == "" {
		cfg.TaskQueue = defaultTaskQueue
	}
	return &Engine{
		client:     c,
		config:     cfg,
		workflows:  make(map[string]interface{}),
		ownsClient: ownsClient,
	}
}

// Close closes the Temporal client when this engine owns it.
func (e *Engine) Close() {
	if e.ownsClient && e.client != nil {
		e.client.Close()
	}
}

// RegisterWorkflowType registers a workflow function type for execution.
func (e *Engine) RegisterWorkflowType(name string, workflowFunc interface{}) {
	e.workflows[name] = workflowFunc
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	_ = ctx
	// Temporal workflows are registered via worker, not the engine
	e.workflows[def.ID] = def
	return nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	_ = ctx
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
	workflowID, runID := splitExecutionID(executionID)
	resp, err := e.client.DescribeWorkflowExecution(ctx, workflowID, runID)
	if err != nil {
		return nil, pkgerrors.NotFound("execution not found", err)
	}

	info := resp.WorkflowExecutionInfo
	exec := &workflow.Execution{
		ID:         info.Execution.RunId,
		WorkflowID: info.Execution.WorkflowId,
		Status:     MapTemporalStatus(info.Status),
	}

	if info.StartTime != nil {
		exec.StartedAt = info.StartTime.AsTime()
	}
	if info.CloseTime != nil {
		exec.CompletedAt = info.CloseTime.AsTime()
	}

	return exec, nil
}

// MapTemporalStatus maps Temporal SDK enums to workflow.ExecutionStatus.
func MapTemporalStatus(status enumspb.WorkflowExecutionStatus) workflow.ExecutionStatus {
	switch status {
	case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return workflow.StatusRunning
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return workflow.StatusCompleted
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
		return workflow.StatusFailed
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return workflow.StatusCancelled
	case enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return workflow.StatusCancelled
	case enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		return workflow.StatusRunning
	case enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		return workflow.StatusTimedOut
	case enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED:
		return workflow.StatusPending
	default:
		return workflow.StatusPending
	}
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	req := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: e.config.Namespace,
	}
	if opts.Limit > 0 {
		req.PageSize = int32(opts.Limit)
	}
	if opts.PageToken != "" {
		tok, err := base64.StdEncoding.DecodeString(opts.PageToken)
		if err != nil {
			// Accept raw tokens for callers that already hold opaque bytes as string.
			req.NextPageToken = []byte(opts.PageToken)
		} else {
			req.NextPageToken = tok
		}
	}

	var parts []string
	if opts.WorkflowID != "" {
		parts = append(parts, fmt.Sprintf(`WorkflowId = "%s"`, escapeVisibility(opts.WorkflowID)))
	}
	if opts.Status != "" {
		if st := visibilityStatus(opts.Status); st != "" {
			parts = append(parts, fmt.Sprintf(`ExecutionStatus = "%s"`, st))
		}
	}
	if len(parts) > 0 {
		req.Query = strings.Join(parts, " AND ")
	}

	resp, err := e.client.ListWorkflow(ctx, req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list Temporal executions", err)
	}

	result := &workflow.ListResult{
		Executions: make([]*workflow.Execution, 0, len(resp.Executions)),
	}
	for _, info := range resp.Executions {
		if info == nil || info.Execution == nil {
			continue
		}
		exec := &workflow.Execution{
			ID:         info.Execution.RunId,
			WorkflowID: info.Execution.WorkflowId,
			Status:     MapTemporalStatus(info.Status),
		}
		if info.StartTime != nil {
			exec.StartedAt = info.StartTime.AsTime()
		}
		if info.CloseTime != nil {
			exec.CompletedAt = info.CloseTime.AsTime()
		}
		result.Executions = append(result.Executions, exec)
	}
	if len(resp.NextPageToken) > 0 {
		result.NextPageToken = base64.StdEncoding.EncodeToString(resp.NextPageToken)
	}
	return result, nil
}

func visibilityStatus(status workflow.ExecutionStatus) string {
	switch status {
	case workflow.StatusRunning:
		return "Running"
	case workflow.StatusCompleted:
		return "Completed"
	case workflow.StatusFailed:
		return "Failed"
	case workflow.StatusCancelled:
		return "Canceled"
	case workflow.StatusTimedOut:
		return "TimedOut"
	case workflow.StatusPending:
		return ""
	default:
		return ""
	}
}

func escapeVisibility(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func splitExecutionID(executionID string) (workflowID, runID string) {
	// Accept "workflowID/runID" or bare workflow/run IDs.
	if i := strings.IndexByte(executionID, '/'); i > 0 {
		return executionID[:i], executionID[i+1:]
	}
	return executionID, ""
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	workflowID, runID := splitExecutionID(executionID)
	err := e.client.CancelWorkflow(ctx, workflowID, runID)
	if err != nil {
		return pkgerrors.Internal("failed to cancel workflow", err)
	}
	return nil
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	workflowID, runID := splitExecutionID(executionID)
	err := e.client.SignalWorkflow(ctx, workflowID, runID, signalName, data)
	if err != nil {
		return pkgerrors.Internal("failed to signal workflow", err)
	}
	return nil
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	workflowID, runID := splitExecutionID(executionID)
	run := e.client.GetWorkflow(ctx, workflowID, runID)

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
