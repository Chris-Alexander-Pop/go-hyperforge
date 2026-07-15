package saga

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// DurableExecutor runs sagas with persisted state and crash resume.
type DurableExecutor struct {
	registry *SagaRegistry
	store    StateStore
}

// NewDurableExecutor creates an executor backed by store.
// registry must contain saga definitions used for Execute/Resume.
func NewDurableExecutor(registry *SagaRegistry, store StateStore) *DurableExecutor {
	return &DurableExecutor{registry: registry, store: store}
}

// Execute starts a new durable saga run and persists progress after each step.
func (d *DurableExecutor) Execute(ctx context.Context, sagaName string, input any) (*Execution, error) {
	if d.store == nil {
		return nil, errors.InvalidArgument("saga state store is required", nil)
	}
	s, ok := d.registry.Get(sagaName)
	if !ok {
		return nil, errors.NotFound("saga not found: "+sagaName, nil)
	}

	now := time.Now().UTC()
	state := &PersistedState{
		ID:            uuid.NewString(),
		SagaName:      sagaName,
		Status:        StatusRunning,
		NextStepIndex: 0,
		Input:         input,
		CurrentData:   input,
		Steps:         make([]PersistedStepResult, 0, len(s.steps)),
		StartedAt:     now,
		UpdatedAt:     now,
	}
	if err := d.store.Save(ctx, state); err != nil {
		return nil, errors.Internal("failed to persist saga state", err)
	}

	return d.runFrom(ctx, s, state)
}

// Resume continues a previously persisted execution after a crash or restart.
func (d *DurableExecutor) Resume(ctx context.Context, executionID string) (*Execution, error) {
	if d.store == nil {
		return nil, errors.InvalidArgument("saga state store is required", nil)
	}
	state, err := d.store.Load(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if state.IsTerminal() {
		return stateToExecution(state), nil
	}

	s, ok := d.registry.Get(state.SagaName)
	if !ok {
		return nil, errors.NotFound("saga not found: "+state.SagaName, nil)
	}

	if state.Status == StatusCompensating {
		if err := d.compensateDurable(ctx, s, state); err != nil {
			return stateToExecution(state), err
		}
		return stateToExecution(state), nil
	}

	state.Status = StatusRunning
	state.UpdatedAt = time.Now().UTC()
	if err := d.store.Save(ctx, state); err != nil {
		return nil, errors.Internal("failed to persist saga state", err)
	}
	return d.runFrom(ctx, s, state)
}

// ResumeAll resumes every incomplete execution found in the store.
func (d *DurableExecutor) ResumeAll(ctx context.Context) ([]*Execution, error) {
	incomplete, err := d.store.ListIncomplete(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*Execution, 0, len(incomplete))
	for _, st := range incomplete {
		exec, err := d.Resume(ctx, st.ID)
		if exec != nil {
			out = append(out, exec)
		}
		if err != nil && exec == nil {
			return out, err
		}
	}
	return out, nil
}

func (d *DurableExecutor) runFrom(ctx context.Context, s *Saga, state *PersistedState) (*Execution, error) {
	data := state.CurrentData
	steps := s.Steps()

	for i := state.NextStepIndex; i < len(steps); i++ {
		if err := ctx.Err(); err != nil {
			state.Status = StatusRunning
			state.UpdatedAt = time.Now().UTC()
			_ = d.store.Save(ctx, state)
			return stateToExecution(state), err
		}

		step := steps[i]
		stepResult := PersistedStepResult{
			Name:      step.Name,
			Status:    StatusRunning,
			StartedAt: time.Now().UTC(),
		}

		var output any
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
		stepResult.CompletedAt = time.Now().UTC()

		if err != nil {
			stepResult.Status = StatusFailed
			stepResult.Error = err.Error()
			state.Steps = append(state.Steps, stepResult)
			state.Status = StatusCompensating
			state.Error = err.Error()
			state.UpdatedAt = time.Now().UTC()
			_ = d.store.Save(ctx, state)

			if compErr := d.compensateDurable(ctx, s, state); compErr != nil {
				return stateToExecution(state), err
			}
			return stateToExecution(state), err
		}

		stepResult.Status = StatusCompleted
		stepResult.Output = output
		state.Steps = append(state.Steps, stepResult)
		state.CurrentData = output
		state.NextStepIndex = i + 1
		state.UpdatedAt = time.Now().UTC()
		if saveErr := d.store.Save(ctx, state); saveErr != nil {
			return stateToExecution(state), errors.Internal("failed to persist saga state", saveErr)
		}
		data = output
	}

	now := time.Now().UTC()
	state.Status = StatusCompleted
	state.Output = data
	state.CompletedAt = now
	state.UpdatedAt = now
	if err := d.store.Save(ctx, state); err != nil {
		return stateToExecution(state), errors.Internal("failed to persist saga state", err)
	}
	return stateToExecution(state), nil
}

func (d *DurableExecutor) compensateDurable(ctx context.Context, s *Saga, state *PersistedState) error {
	steps := s.Steps()
	stepByName := make(map[string]Step, len(steps))
	for _, st := range steps {
		stepByName[st.Name] = st
	}

	var firstErr error
	for i := len(state.Steps) - 1; i >= 0; i-- {
		sr := &state.Steps[i]
		if sr.Status != StatusCompleted {
			continue
		}
		step, ok := stepByName[sr.Name]
		if !ok || step.Compensate == nil {
			sr.Status = StatusCompensated
			continue
		}
		if _, err := step.Compensate(ctx, sr.Output); err != nil {
			sr.Status = StatusFailed
			sr.Error = err.Error()
			if firstErr == nil {
				firstErr = err
			}
		} else {
			sr.Status = StatusCompensated
		}
	}

	now := time.Now().UTC()
	if firstErr != nil {
		state.Status = StatusFailed
		if state.Error == "" {
			state.Error = firstErr.Error()
		} else {
			state.Error = state.Error + "; compensation failed: " + firstErr.Error()
		}
	} else {
		state.Status = StatusCompensated
	}
	state.CompletedAt = now
	state.UpdatedAt = now
	_ = d.store.Save(ctx, state)
	return firstErr
}

func stateToExecution(state *PersistedState) *Execution {
	if state == nil {
		return nil
	}
	exec := &Execution{
		ID:          state.ID,
		SagaName:    state.SagaName,
		Status:      state.Status,
		Input:       state.Input,
		Output:      state.Output,
		Error:       state.Error,
		StartedAt:   state.StartedAt,
		CompletedAt: state.CompletedAt,
		Steps:       make([]*StepResult, 0, len(state.Steps)),
	}
	for _, sr := range state.Steps {
		r := &StepResult{
			Name:        sr.Name,
			Status:      sr.Status,
			Output:      sr.Output,
			StartedAt:   sr.StartedAt,
			CompletedAt: sr.CompletedAt,
		}
		if sr.Error != "" {
			r.Error = errors.Internal(sr.Error, nil)
		}
		exec.Steps = append(exec.Steps, r)
	}
	return exec
}
