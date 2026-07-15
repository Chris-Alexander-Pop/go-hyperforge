package workflow

import (
	"context"
	"time"
)

// Driver constants for workflow backends.
const (
	DriverMemory        = "memory"
	DriverStepFunctions = "stepfunctions"
	DriverTemporal      = "temporal"
	DriverLogicApps     = "logicapps"
)

// ExecutionStatus represents the status of a workflow execution.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusTimedOut  ExecutionStatus = "timed_out"
)

// Config holds configuration for the workflow engine.
type Config struct {
	// Driver specifies the workflow backend.
	Driver string `env:"WORKFLOW_DRIVER" env-default:"memory"`

	// AWS Step Functions specific
	AWSAccessKeyID     string `env:"WORKFLOW_AWS_ACCESS_KEY"`
	AWSSecretAccessKey string `env:"WORKFLOW_AWS_SECRET_KEY"`
	AWSRegion          string `env:"WORKFLOW_AWS_REGION" env-default:"us-east-1"`

	// Temporal specific
	TemporalHost      string `env:"TEMPORAL_HOST" env-default:"localhost:7233"`
	TemporalNamespace string `env:"TEMPORAL_NAMESPACE" env-default:"default"`

	// Azure Logic Apps specific
	AzureSubscriptionID string `env:"WORKFLOW_AZURE_SUBSCRIPTION"`
	AzureResourceGroup  string `env:"WORKFLOW_AZURE_RESOURCE_GROUP"`

	// Common options
	DefaultTimeout time.Duration `env:"WORKFLOW_TIMEOUT" env-default:"1h"`
}

// WorkflowDefinition defines a workflow structure.
type WorkflowDefinition struct {
	// ID is the unique identifier.
	ID string

	// Name is the workflow name.
	Name string

	// Version is the workflow version.
	Version string

	// States defines the workflow states/steps.
	States []State

	// StartAt is the initial state.
	StartAt string

	// TimeoutSeconds is the workflow timeout.
	TimeoutSeconds int

	// CreatedAt is when the workflow was created.
	CreatedAt time.Time
}

// State represents a workflow state/step.
type State struct {
	// Name is the state name.
	Name string

	// Type is the state type (Task, Choice, Wait, Parallel, etc.).
	Type string

	// Resource is the resource to invoke (Lambda ARN, activity name, etc.).
	Resource string

	// Next is the next state.
	Next string

	// End marks this as an end state.
	End bool

	// Retry configures retry behavior.
	Retry []RetryPolicy

	// Catch configures error handling.
	Catch []CatchPolicy

	// TimeoutSeconds is the state timeout.
	TimeoutSeconds int
}

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	// ErrorEquals are errors to retry on.
	ErrorEquals []string

	// IntervalSeconds is the initial retry interval.
	IntervalSeconds int

	// MaxAttempts is the maximum retry attempts.
	MaxAttempts int

	// BackoffRate is the backoff multiplier.
	BackoffRate float64
}

// CatchPolicy defines error handling.
type CatchPolicy struct {
	// ErrorEquals are errors to catch.
	ErrorEquals []string

	// Next is the state to transition to.
	Next string

	// ResultPath stores the error.
	ResultPath string
}

// Execution represents a workflow execution.
type Execution struct {
	// ID is the unique execution identifier.
	ID string

	// WorkflowID is the workflow being executed.
	WorkflowID string

	// Status is the current status.
	Status ExecutionStatus

	// Input is the workflow input.
	Input interface{}

	// Output is the workflow output (if completed).
	Output interface{}

	// Error is the error message (if failed).
	Error string

	// CurrentState is the current state name.
	CurrentState string

	// StartedAt is when execution started.
	StartedAt time.Time

	// CompletedAt is when execution completed.
	CompletedAt time.Time
}

// StartOptions configures workflow execution.
type StartOptions struct {
	// WorkflowID is the workflow definition ID.
	WorkflowID string

	// ExecutionID is a custom execution ID (auto-generated if empty).
	ExecutionID string

	// Input is the workflow input data.
	Input interface{}

	// Timeout overrides the default timeout.
	Timeout time.Duration

	// IdempotencyKey prevents duplicate executions.
	IdempotencyKey string
}

// ListOptions configures execution listing.
type ListOptions struct {
	// WorkflowID filters by workflow.
	WorkflowID string

	// Status filters by status.
	Status ExecutionStatus

	// Limit is the maximum results.
	Limit int

	// PageToken is for pagination.
	PageToken string
}

// ListResult contains the list result.
type ListResult struct {
	// Executions is the list of executions.
	Executions []*Execution

	// NextPageToken is for pagination.
	NextPageToken string
}

// WorkflowEngine defines the interface for workflow orchestration.
type WorkflowEngine interface {
	// RegisterWorkflow registers a workflow definition.
	RegisterWorkflow(ctx context.Context, def WorkflowDefinition) error

	// GetWorkflow retrieves a workflow definition.
	GetWorkflow(ctx context.Context, workflowID string) (*WorkflowDefinition, error)

	// Start starts a new workflow execution.
	Start(ctx context.Context, opts StartOptions) (*Execution, error)

	// GetExecution retrieves an execution by ID.
	GetExecution(ctx context.Context, executionID string) (*Execution, error)

	// ListExecutions returns executions matching the options.
	ListExecutions(ctx context.Context, opts ListOptions) (*ListResult, error)

	// Cancel cancels a running execution.
	Cancel(ctx context.Context, executionID string) error

	// Signal sends a signal to a running execution.
	Signal(ctx context.Context, executionID string, signalName string, data interface{}) error

	// Wait waits for an execution to complete.
	Wait(ctx context.Context, executionID string) (*Execution, error)
}
