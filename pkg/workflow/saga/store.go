package saga

import (
	"context"
	"time"
)

// StateStore persists saga execution state for crash recovery.
type StateStore interface {
	// Save upserts execution state.
	Save(ctx context.Context, state *PersistedState) error

	// Load retrieves execution state by ID.
	Load(ctx context.Context, executionID string) (*PersistedState, error)

	// Delete removes execution state.
	Delete(ctx context.Context, executionID string) error

	// ListIncomplete returns executions that are not terminal
	// (pending, running, or compensating).
	ListIncomplete(ctx context.Context) ([]*PersistedState, error)
}

// PersistedStepResult is a JSON-friendly step result snapshot.
type PersistedStepResult struct {
	Name        string          `json:"name"`
	Status      ExecutionStatus `json:"status"`
	Output      any             `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at,omitempty"`
}

// PersistedState is durable saga execution state.
//
// NextStepIndex is the index of the next step to run. After a crash mid-step,
// Resume re-executes that step (at-least-once semantics).
type PersistedState struct {
	ID            string                `json:"id"`
	SagaName      string                `json:"saga_name"`
	Status        ExecutionStatus       `json:"status"`
	NextStepIndex int                   `json:"next_step_index"`
	Input         any                   `json:"input,omitempty"`
	CurrentData   any                   `json:"current_data,omitempty"`
	Output        any                   `json:"output,omitempty"`
	Error         string                `json:"error,omitempty"`
	Steps         []PersistedStepResult `json:"steps,omitempty"`
	StartedAt     time.Time             `json:"started_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
	CompletedAt   time.Time             `json:"completed_at,omitempty"`
}

// IsTerminal reports whether the execution has finished.
func (s *PersistedState) IsTerminal() bool {
	switch s.Status {
	case StatusCompleted, StatusCompensated, StatusFailed:
		return true
	default:
		return false
	}
}

// Clone returns a deep-ish copy safe for callers to mutate step slices.
func (s *PersistedState) Clone() *PersistedState {
	if s == nil {
		return nil
	}
	cp := *s
	if s.Steps != nil {
		cp.Steps = make([]PersistedStepResult, len(s.Steps))
		copy(cp.Steps, s.Steps)
	}
	return &cp
}
