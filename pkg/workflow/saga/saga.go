package saga

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// ExecutionStatus represents saga execution status.
type ExecutionStatus string

const (
	StatusPending      ExecutionStatus = "pending"
	StatusRunning      ExecutionStatus = "running"
	StatusCompleted    ExecutionStatus = "completed"
	StatusCompensating ExecutionStatus = "compensating"
	StatusCompensated  ExecutionStatus = "compensated"
	StatusFailed       ExecutionStatus = "failed"
)

// ActionFunc is the function signature for saga actions.
type ActionFunc func(ctx context.Context, data interface{}) (interface{}, error)

// Step represents a saga step with action and compensation.
type Step struct {
	// Name is the step name.
	Name string

	// Action is the forward action.
	Action ActionFunc

	// Compensate is the compensation (rollback) action.
	Compensate ActionFunc

	// Timeout is the step timeout.
	Timeout time.Duration
}

// StepResult contains the result of a step execution.
type StepResult struct {
	// Name is the step name.
	Name string

	// Status is the step status.
	Status ExecutionStatus

	// Output is the step output.
	Output interface{}

	// Error is the step error (if any).
	Error error

	// StartedAt is when the step started.
	StartedAt time.Time

	// CompletedAt is when the step completed.
	CompletedAt time.Time
}

// Execution represents a saga execution.
type Execution struct {
	// ID is the unique execution ID.
	ID string

	// SagaName is the saga being executed.
	SagaName string

	// Status is the current status.
	Status ExecutionStatus

	// Input is the initial input.
	Input interface{}

	// Output is the final output.
	Output interface{}

	// Error is the error message (if failed).
	Error string

	// Steps contains step results.
	Steps []*StepResult

	// StartedAt is when execution started.
	StartedAt time.Time

	// CompletedAt is when execution completed.
	CompletedAt time.Time
}

// Saga represents a saga orchestrator.
type Saga struct {
	name  string
	steps []Step
}

// New creates a new saga.
func New(name string) *Saga {
	return &Saga{
		name:  name,
		steps: []Step{},
	}
}

// AddStep adds a step to the saga.
func (s *Saga) AddStep(step Step) *Saga {
	s.steps = append(s.steps, step)
	return s
}

// Execute runs the saga with the given input.
func (s *Saga) Execute(ctx context.Context, input interface{}) (*Execution, error) {
	exec := &Execution{
		ID:        uuid.NewString(),
		SagaName:  s.name,
		Status:    StatusRunning,
		Input:     input,
		Steps:     make([]*StepResult, 0, len(s.steps)),
		StartedAt: time.Now(),
	}

	data := input
	completedSteps := make([]*StepResult, 0)

	for _, step := range s.steps {
		stepResult := &StepResult{
			Name:      step.Name,
			Status:    StatusRunning,
			StartedAt: time.Now(),
		}
		exec.Steps = append(exec.Steps, stepResult)

		var output interface{}
		err := func() error {
			stepCtx := ctx
			if step.Timeout > 0 {
				var cancel context.CancelFunc
				stepCtx, cancel = context.WithTimeout(ctx, step.Timeout)
				defer cancel()
			}

			var innerErr error
			output, innerErr = step.Action(stepCtx, data)
			return innerErr
		}()
		stepResult.CompletedAt = time.Now()

		if err != nil {
			stepResult.Status = StatusFailed
			stepResult.Error = err
			exec.Status = StatusCompensating
			exec.Error = err.Error()

			if compErr := s.compensate(ctx, completedSteps); compErr != nil {
				exec.Status = StatusFailed
				exec.Error = exec.Error + "; compensation failed: " + compErr.Error()
			} else {
				exec.Status = StatusCompensated
			}

			exec.CompletedAt = time.Now()
			return exec, err
		}

		stepResult.Status = StatusCompleted
		stepResult.Output = output
		completedSteps = append(completedSteps, stepResult)
		data = output
	}

	exec.Status = StatusCompleted
	exec.Output = data
	exec.CompletedAt = time.Now()

	return exec, nil
}

func (s *Saga) compensate(ctx context.Context, completedSteps []*StepResult) error {
	var errs []error

	for i := len(completedSteps) - 1; i >= 0; i-- {
		stepResult := completedSteps[i]

		var step *Step
		for j := range s.steps {
			if s.steps[j].Name == stepResult.Name {
				step = &s.steps[j]
				break
			}
		}

		if step == nil || step.Compensate == nil {
			continue
		}

		if _, err := step.Compensate(ctx, stepResult.Output); err != nil {
			errs = append(errs, err)
			stepResult.Status = StatusFailed
			stepResult.Error = err
		} else {
			stepResult.Status = StatusCompensated
		}
	}

	if len(errs) > 0 {
		return errors.Internal("compensation failed", stderrors.Join(errs...))
	}

	return nil
}

// Name returns the saga name.
func (s *Saga) Name() string {
	return s.name
}

// Steps returns a copy of the saga steps.
func (s *Saga) Steps() []Step {
	out := make([]Step, len(s.steps))
	copy(out, s.steps)
	return out
}

// SagaRegistry manages saga definitions.
type SagaRegistry struct {
	mu    *concurrency.SmartRWMutex
	sagas map[string]*Saga
}

// NewRegistry creates a new saga registry.
func NewRegistry() *SagaRegistry {
	return &SagaRegistry{
		mu:    concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "workflow-saga-registry"}),
		sagas: make(map[string]*Saga),
	}
}

// Register registers a saga.
func (r *SagaRegistry) Register(saga *Saga) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sagas[saga.name] = saga
}

// Get retrieves a saga by name.
func (r *SagaRegistry) Get(name string) (*Saga, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	saga, ok := r.sagas[name]
	return saga, ok
}
