package memory

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/workflow"
	"github.com/google/uuid"
)

// copyExecution creates a shallow copy of the execution to avoid data races
// when returning the object to callers while the engine might be updating it.
func copyExecution(src *workflow.Execution) *workflow.Execution {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

// Engine implements an in-memory workflow engine for testing.
type Engine struct {
	mu         sync.RWMutex
	workflows  map[string]*workflow.WorkflowDefinition
	executions map[string]*workflow.Execution
	signals    map[string]map[string]interface{} // execID -> signalName -> data
	waiters    map[string][]chan *workflow.Execution
	config     workflow.Config
}

// New creates a new in-memory workflow engine.
func New() workflow.WorkflowEngine {
	return &Engine{
		workflows:  make(map[string]*workflow.WorkflowDefinition),
		executions: make(map[string]*workflow.Execution),
		signals:    make(map[string]map[string]interface{}),
		waiters:    make(map[string][]chan *workflow.Execution),
		config:     workflow.Config{DefaultTimeout: time.Hour},
	}
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	def.CreatedAt = time.Now()

	e.workflows[def.ID] = &def
	return nil
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	wf, ok := e.workflows[workflowID]
	if !ok {
		return nil, errors.NotFound("workflow not found", nil)
	}

	return wf, nil
}

func (e *Engine) Start(ctx context.Context, opts workflow.StartOptions) (*workflow.Execution, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check workflow exists
	if _, ok := e.workflows[opts.WorkflowID]; !ok {
		return nil, errors.NotFound("workflow not found", nil)
	}

	execID := opts.ExecutionID
	if execID == "" {
		execID = uuid.NewString()
	}

	// Check for duplicate execution ID
	if _, exists := e.executions[execID]; exists {
		return nil, errors.Conflict("execution already exists", nil)
	}

	exec := &workflow.Execution{
		ID:         execID,
		WorkflowID: opts.WorkflowID,
		Status:     workflow.StatusRunning,
		Input:      opts.Input,
		StartedAt:  time.Now(),
	}

	e.executions[execID] = exec
	e.signals[execID] = make(map[string]interface{})

	// Simulate async execution
	go e.simulateExecution(ctx, exec, opts.Timeout)

	return copyExecution(exec), nil
}

func (e *Engine) simulateExecution(ctx context.Context, exec *workflow.Execution, timeout time.Duration) {
	if timeout == 0 {
		_ = e.config.DefaultTimeout
	}

	// Simulate some work
	select {
	case <-time.After(100 * time.Millisecond):
		e.mu.Lock()
		exec.Status = workflow.StatusCompleted
		exec.Output = exec.Input
		exec.CompletedAt = time.Now()

		// Notify waiters
		for _, ch := range e.waiters[exec.ID] {
			select {
			case ch <- exec:
			default:
			}
		}
		delete(e.waiters, exec.ID)
		e.mu.Unlock()

	case <-ctx.Done():
		e.mu.Lock()
		exec.Status = workflow.StatusCancelled
		exec.CompletedAt = time.Now()
		e.mu.Unlock()
	}
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return nil, errors.NotFound("execution not found", nil)
	}

	return copyExecution(exec), nil
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := &workflow.ListResult{
		Executions: make([]*workflow.Execution, 0),
	}

	for _, exec := range e.executions {
		// Apply filters
		if opts.WorkflowID != "" && exec.WorkflowID != opts.WorkflowID {
			continue
		}
		if opts.Status != "" && exec.Status != opts.Status {
			continue
		}
		result.Executions = append(result.Executions, copyExecution(exec))
	}

	// Apply limit
	if opts.Limit > 0 && len(result.Executions) > opts.Limit {
		result.Executions = result.Executions[:opts.Limit]
		result.NextPageToken = "more"
	}

	return result, nil
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return errors.NotFound("execution not found", nil)
	}

	if exec.Status != workflow.StatusRunning {
		return errors.Conflict("execution is not running", nil)
	}

	exec.Status = workflow.StatusCancelled
	exec.CompletedAt = time.Now()

	return nil
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return errors.NotFound("execution not found", nil)
	}

	if exec.Status != workflow.StatusRunning {
		return errors.Conflict("execution is not running", nil)
	}

	e.signals[executionID][signalName] = data
	return nil
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	e.mu.Lock()
	exec, ok := e.executions[executionID]
	if !ok {
		e.mu.Unlock()
		return nil, errors.NotFound("execution not found", nil)
	}

	// Already completed
	if exec.Status != workflow.StatusRunning && exec.Status != workflow.StatusPending {
		e.mu.Unlock()
		return copyExecution(exec), nil
	}

	// Create waiter
	ch := make(chan *workflow.Execution, 1)
	e.waiters[executionID] = append(e.waiters[executionID], ch)
	e.mu.Unlock()

	select {
	case result := <-ch:
		return copyExecution(result), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
