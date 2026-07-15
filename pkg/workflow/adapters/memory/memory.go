package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
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
	mu         *concurrency.SmartRWMutex
	workflows  map[string]*workflow.WorkflowDefinition
	executions map[string]*workflow.Execution
	signals    map[string]map[string]interface{} // execID -> signalName -> data
	waiters    map[string][]chan *workflow.Execution
	config     workflow.Config
	// workDuration is how long simulateExecution takes before completing (tests may shorten).
	workDuration time.Duration
}

// New creates a new in-memory workflow engine.
func New() workflow.WorkflowEngine {
	return &Engine{
		mu:           concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-memory"}),
		workflows:    make(map[string]*workflow.WorkflowDefinition),
		executions:   make(map[string]*workflow.Execution),
		signals:      make(map[string]map[string]interface{}),
		waiters:      make(map[string][]chan *workflow.Execution),
		config:       workflow.Config{DefaultTimeout: time.Hour},
		workDuration: 100 * time.Millisecond,
	}
}

// NewWithConfig creates an engine with an explicit config (e.g. DefaultTimeout).
func NewWithConfig(cfg workflow.Config) workflow.WorkflowEngine {
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = time.Hour
	}
	return &Engine{
		mu:           concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-memory"}),
		workflows:    make(map[string]*workflow.WorkflowDefinition),
		executions:   make(map[string]*workflow.Execution),
		signals:      make(map[string]map[string]interface{}),
		waiters:      make(map[string][]chan *workflow.Execution),
		config:       cfg,
		workDuration: 100 * time.Millisecond,
	}
}

func (e *Engine) notifyWaitersLocked(exec *workflow.Execution) {
	for _, ch := range e.waiters[exec.ID] {
		select {
		case ch <- copyExecution(exec):
		default:
		}
	}
	delete(e.waiters, exec.ID)
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
		return nil, workflow.ErrWorkflowNotFound
	}

	cp := *wf
	return &cp, nil
}

func (e *Engine) Start(ctx context.Context, opts workflow.StartOptions) (*workflow.Execution, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.workflows[opts.WorkflowID]; !ok {
		return nil, workflow.ErrWorkflowNotFound
	}

	execID := opts.ExecutionID
	if execID == "" {
		execID = uuid.NewString()
	}

	if _, exists := e.executions[execID]; exists {
		return nil, workflow.ErrExecutionAlreadyExists
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

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = e.config.DefaultTimeout
	}

	go e.simulateExecution(ctx, exec, timeout)

	return copyExecution(exec), nil
}

func (e *Engine) simulateExecution(parent context.Context, exec *workflow.Execution, timeout time.Duration) {
	ctx := parent
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(parent, timeout)
		defer cancel()
	}

	work := e.workDuration
	if work <= 0 {
		work = 100 * time.Millisecond
	}

	select {
	case <-time.After(work):
		e.mu.Lock()
		if exec.Status == workflow.StatusRunning {
			exec.Status = workflow.StatusCompleted
			exec.Output = exec.Input
			exec.CompletedAt = time.Now()
			e.notifyWaitersLocked(exec)
		}
		e.mu.Unlock()

	case <-ctx.Done():
		e.mu.Lock()
		if exec.Status == workflow.StatusRunning {
			// Prefer timeout when our deadline fired; otherwise treat as cancel.
			if ctx.Err() == context.DeadlineExceeded {
				exec.Status = workflow.StatusTimedOut
				exec.Error = workflow.ErrExecutionTimeout.Error()
			} else {
				exec.Status = workflow.StatusCancelled
			}
			exec.CompletedAt = time.Now()
			e.notifyWaitersLocked(exec)
		}
		e.mu.Unlock()
	}
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return nil, workflow.ErrExecutionNotFound
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
		if opts.WorkflowID != "" && exec.WorkflowID != opts.WorkflowID {
			continue
		}
		if opts.Status != "" && exec.Status != opts.Status {
			continue
		}
		result.Executions = append(result.Executions, copyExecution(exec))
	}

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
		return workflow.ErrExecutionNotFound
	}

	if exec.Status != workflow.StatusRunning {
		return workflow.ErrExecutionNotRunning
	}

	exec.Status = workflow.StatusCancelled
	exec.CompletedAt = time.Now()
	e.notifyWaitersLocked(exec)

	return nil
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.executions[executionID]
	if !ok {
		return workflow.ErrExecutionNotFound
	}

	if exec.Status != workflow.StatusRunning {
		return workflow.ErrExecutionNotRunning
	}

	e.signals[executionID][signalName] = data
	return nil
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	e.mu.Lock()
	exec, ok := e.executions[executionID]
	if !ok {
		e.mu.Unlock()
		return nil, workflow.ErrExecutionNotFound
	}

	if exec.Status != workflow.StatusRunning && exec.Status != workflow.StatusPending {
		e.mu.Unlock()
		return copyExecution(exec), nil
	}

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
