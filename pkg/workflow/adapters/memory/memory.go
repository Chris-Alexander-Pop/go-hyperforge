package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
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
	handlers   map[string]workflow.TaskHandler // resource -> handler
	idempo     map[string]string               // idempotencyKey -> executionID
	config     workflow.Config
	// workDuration is used when a workflow has no StartAt/states (legacy pass-through).
	workDuration time.Duration
}

// New creates a new in-memory workflow engine.
func New() workflow.WorkflowEngine {
	return newEngine(workflow.Config{DefaultTimeout: time.Hour})
}

// NewWithConfig creates an engine with an explicit config (e.g. DefaultTimeout).
func NewWithConfig(cfg workflow.Config) workflow.WorkflowEngine {
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = time.Hour
	}
	return newEngine(cfg)
}

func newEngine(cfg workflow.Config) *Engine {
	return &Engine{
		mu:           concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-memory"}),
		workflows:    make(map[string]*workflow.WorkflowDefinition),
		executions:   make(map[string]*workflow.Execution),
		signals:      make(map[string]map[string]interface{}),
		waiters:      make(map[string][]chan *workflow.Execution),
		handlers:     make(map[string]workflow.TaskHandler),
		idempo:       make(map[string]string),
		config:       cfg,
		workDuration: 100 * time.Millisecond,
	}
}

// RegisterTaskHandler registers a Task state handler for a Resource name.
// Safe to call on the concrete *Engine (New returns WorkflowEngine — type-assert when needed).
func (e *Engine) RegisterTaskHandler(resource string, h workflow.TaskHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[resource] = h
}

// SetWorkDuration overrides the legacy empty-workflow work duration (tests).
func (e *Engine) SetWorkDuration(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.workDuration = d
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

	wf, ok := e.workflows[opts.WorkflowID]
	if !ok {
		return nil, workflow.ErrWorkflowNotFound
	}

	if opts.IdempotencyKey != "" {
		if existingID, hit := e.idempo[opts.IdempotencyKey]; hit {
			if exec, exists := e.executions[existingID]; exists {
				return copyExecution(exec), nil
			}
		}
	}

	execID := opts.ExecutionID
	if execID == "" {
		execID = uuid.NewString()
	}

	if _, exists := e.executions[execID]; exists {
		return nil, workflow.ErrExecutionAlreadyExists
	}

	startAt := wf.StartAt
	exec := &workflow.Execution{
		ID:           execID,
		WorkflowID:   opts.WorkflowID,
		Status:       workflow.StatusRunning,
		Input:        opts.Input,
		CurrentState: startAt,
		StartedAt:    time.Now(),
	}

	e.executions[execID] = exec
	e.signals[execID] = make(map[string]interface{})
	if opts.IdempotencyKey != "" {
		e.idempo[opts.IdempotencyKey] = execID
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = e.config.DefaultTimeout
	}
	if wf.TimeoutSeconds > 0 {
		wfTimeout := time.Duration(wf.TimeoutSeconds) * time.Second
		if timeout <= 0 || wfTimeout < timeout {
			timeout = wfTimeout
		}
	}

	// Snapshot definition + handlers for the runner goroutine.
	defCopy := *wf
	handlers := make(map[string]workflow.TaskHandler, len(e.handlers))
	for k, v := range e.handlers {
		handlers[k] = v
	}
	work := e.workDuration

	go e.runExecution(ctx, exec, &defCopy, handlers, timeout, work)

	return copyExecution(exec), nil
}

func (e *Engine) runExecution(
	parent context.Context,
	exec *workflow.Execution,
	def *workflow.WorkflowDefinition,
	handlers map[string]workflow.TaskHandler,
	timeout time.Duration,
	workDuration time.Duration,
) {
	ctx := parent
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	// Legacy path: no state machine defined — sleep then echo input (keeps existing tests).
	if def.StartAt == "" || len(def.States) == 0 {
		e.simulatePassThrough(ctx, exec, workDuration)
		return
	}

	states := make(map[string]workflow.State, len(def.States))
	for _, s := range def.States {
		states[s.Name] = s
	}

	data := exec.Input
	current := def.StartAt

	for {
		if ctx.Err() != nil {
			e.finish(exec, terminalFromCtx(ctx), nil, ctx.Err())
			return
		}

		e.mu.Lock()
		if exec.Status != workflow.StatusRunning {
			e.mu.Unlock()
			return
		}
		exec.CurrentState = current
		e.mu.Unlock()

		st, ok := states[current]
		if !ok {
			e.finish(exec, workflow.StatusFailed, nil, fmt.Errorf("unknown state %q", current))
			return
		}

		var err error
		switch strings.ToLower(st.Type) {
		case "task", "":
			data, err = e.execTask(ctx, st, handlers, data)
		case "wait":
			err = e.execWait(ctx, st)
		case "succeed", "pass":
			// no-op; pass data through
		case "fail":
			e.finish(exec, workflow.StatusFailed, data, fmt.Errorf("state %q failed", st.Name))
			return
		default:
			e.finish(exec, workflow.StatusFailed, data, fmt.Errorf("unsupported state type %q", st.Type))
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				e.finish(exec, terminalFromCtx(ctx), data, ctx.Err())
				return
			}
			e.finish(exec, workflow.StatusFailed, data, err)
			return
		}

		if st.End || st.Next == "" {
			e.finish(exec, workflow.StatusCompleted, data, nil)
			return
		}
		current = st.Next
	}
}

func (e *Engine) execTask(ctx context.Context, st workflow.State, handlers map[string]workflow.TaskHandler, input interface{}) (interface{}, error) {
	h := handlers[st.Resource]
	if h == nil && st.Resource != "" {
		// No handler registered: pass-through (useful for structural tests).
		return input, nil
	}
	if h == nil {
		return input, nil
	}
	taskCtx := ctx
	var cancel context.CancelFunc
	if st.TimeoutSeconds > 0 {
		taskCtx, cancel = context.WithTimeout(ctx, time.Duration(st.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	return h(taskCtx, input)
}

func (e *Engine) execWait(ctx context.Context, st workflow.State) error {
	sec := st.Seconds
	if sec <= 0 {
		return nil
	}
	t := time.NewTimer(time.Duration(sec) * time.Second)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) simulatePassThrough(ctx context.Context, exec *workflow.Execution, work time.Duration) {
	if work <= 0 {
		work = 100 * time.Millisecond
	}
	select {
	case <-time.After(work):
		e.finish(exec, workflow.StatusCompleted, exec.Input, nil)
	case <-ctx.Done():
		e.finish(exec, terminalFromCtx(ctx), nil, ctx.Err())
	}
}

func terminalFromCtx(ctx context.Context) workflow.ExecutionStatus {
	if ctx.Err() == context.DeadlineExceeded {
		return workflow.StatusTimedOut
	}
	return workflow.StatusCancelled
}

func (e *Engine) finish(exec *workflow.Execution, status workflow.ExecutionStatus, output interface{}, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if exec.Status != workflow.StatusRunning && exec.Status != workflow.StatusPending {
		return
	}
	exec.Status = status
	exec.CompletedAt = time.Now()
	if output != nil {
		exec.Output = output
	} else if status == workflow.StatusCompleted {
		exec.Output = exec.Input
	}
	if err != nil {
		if status == workflow.StatusTimedOut {
			exec.Error = workflow.ErrExecutionTimeout.Error()
		} else if status == workflow.StatusCancelled {
			// leave Error empty for cancel
		} else {
			exec.Error = err.Error()
		}
	}
	e.notifyWaitersLocked(exec)
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
