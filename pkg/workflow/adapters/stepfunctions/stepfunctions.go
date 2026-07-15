// Package stepfunctions provides an AWS Step Functions adapter for workflow.WorkflowEngine.
//
// This adapter wraps the AWS SDK to manage Step Functions state machines and executions.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/stepfunctions"
//
//	engine, err := stepfunctions.New(stepfunctions.Config{
//	    Region: "us-east-1",
//	    RoleArn: "arn:aws:iam::123456789012:role/StepFunctionsRole",
//	})
//	exec, err := engine.Start(ctx, workflow.StartOptions{WorkflowID: "arn:aws:states:...", Input: data})
package stepfunctions

import (
	"context"
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
)

// Config holds AWS Step Functions configuration.
type Config struct {
	// Region is the AWS region.
	Region string

	// AccessKeyID is the AWS access key.
	AccessKeyID string

	// SecretAccessKey is the AWS secret key.
	SecretAccessKey string

	// Endpoint is an optional custom endpoint (for LocalStack).
	Endpoint string

	// RoleArn is the IAM role ARN used when CreateStateMachine runs (RegisterWorkflow).
	// Required for RegisterWorkflow; Start/Get/List/Cancel do not need it.
	RoleArn string
}

// SFAPI is the Step Functions client surface used by this adapter (for tests).
type SFAPI interface {
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
	DescribeExecution(ctx context.Context, params *sfn.DescribeExecutionInput, optFns ...func(*sfn.Options)) (*sfn.DescribeExecutionOutput, error)
	ListExecutions(ctx context.Context, params *sfn.ListExecutionsInput, optFns ...func(*sfn.Options)) (*sfn.ListExecutionsOutput, error)
	StopExecution(ctx context.Context, params *sfn.StopExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StopExecutionOutput, error)
	SendTaskSuccess(ctx context.Context, params *sfn.SendTaskSuccessInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskSuccessOutput, error)
	SendTaskFailure(ctx context.Context, params *sfn.SendTaskFailureInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskFailureOutput, error)
}

// Engine implements workflow.WorkflowEngine for AWS Step Functions.
type Engine struct {
	client SFAPI
	config Config
}

// New creates a new Step Functions engine.
func New(cfg Config) (*Engine, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, pkgerrors.Internal("failed to load AWS config", err)
	}

	clientOpts := []func(*sfn.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *sfn.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	return NewFromAPI(sfn.NewFromConfig(awsCfg, clientOpts...), cfg), nil
}

// NewFromAPI wraps an SFAPI (SDK client or test double).
func NewFromAPI(api SFAPI, cfg Config) *Engine {
	return &Engine{client: api, config: cfg}
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	if e.config.RoleArn == "" {
		return pkgerrors.InvalidArgument("stepfunctions RoleArn is required to create a state machine", nil)
	}
	definition, err := json.Marshal(map[string]interface{}{
		"StartAt": def.StartAt,
		"States":  convertStates(def.States),
	})
	if err != nil {
		return pkgerrors.Internal("failed to marshal definition", err)
	}

	_, err = e.client.CreateStateMachine(ctx, &sfn.CreateStateMachineInput{
		Name:       aws.String(def.Name),
		Definition: aws.String(string(definition)),
		RoleArn:    aws.String(e.config.RoleArn),
		Type:       types.StateMachineTypeStandard,
	})
	if err != nil {
		return pkgerrors.Internal("failed to create state machine", err)
	}

	return nil
}

func convertStates(states []workflow.State) map[string]interface{} {
	result := make(map[string]interface{})
	for i := range states {
		s := &states[i]
		state := map[string]interface{}{
			"Type": s.Type,
		}
		if s.Resource != "" {
			state["Resource"] = s.Resource
		}
		if s.Next != "" {
			state["Next"] = s.Next
		}
		if s.End {
			state["End"] = true
		}
		result[s.Name] = state
	}
	return result
}

func (e *Engine) GetWorkflow(ctx context.Context, workflowID string) (*workflow.WorkflowDefinition, error) {
	output, err := e.client.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(workflowID),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("state machine not found", err)
	}

	return &workflow.WorkflowDefinition{
		ID:        *output.StateMachineArn,
		Name:      *output.Name,
		CreatedAt: *output.CreationDate,
	}, nil
}

func (e *Engine) Start(ctx context.Context, opts workflow.StartOptions) (*workflow.Execution, error) {
	input := "{}"
	if opts.Input != nil {
		data, err := json.Marshal(opts.Input)
		if err != nil {
			return nil, pkgerrors.InvalidArgument("failed to marshal input", err)
		}
		input = string(data)
	}

	startInput := &sfn.StartExecutionInput{
		StateMachineArn: aws.String(opts.WorkflowID),
		Input:           aws.String(input),
	}

	if opts.ExecutionID != "" {
		startInput.Name = aws.String(opts.ExecutionID)
	}

	output, err := e.client.StartExecution(ctx, startInput)
	if err != nil {
		return nil, pkgerrors.Internal("failed to start execution", err)
	}

	return &workflow.Execution{
		ID:         *output.ExecutionArn,
		WorkflowID: opts.WorkflowID,
		Status:     workflow.StatusRunning,
		Input:      opts.Input,
		StartedAt:  *output.StartDate,
	}, nil
}

func (e *Engine) GetExecution(ctx context.Context, executionID string) (*workflow.Execution, error) {
	output, err := e.client.DescribeExecution(ctx, &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionID),
	})
	if err != nil {
		return nil, pkgerrors.NotFound("execution not found", err)
	}

	exec := &workflow.Execution{
		ID:         *output.ExecutionArn,
		WorkflowID: *output.StateMachineArn,
		Status:     mapStatus(output.Status),
		StartedAt:  *output.StartDate,
	}

	if output.StopDate != nil {
		exec.CompletedAt = *output.StopDate
	}
	if output.Output != nil {
		exec.Output = *output.Output
	}
	if output.Error != nil {
		exec.Error = *output.Error
	}

	return exec, nil
}

func mapStatus(status types.ExecutionStatus) workflow.ExecutionStatus {
	switch status {
	case types.ExecutionStatusRunning:
		return workflow.StatusRunning
	case types.ExecutionStatusSucceeded:
		return workflow.StatusCompleted
	case types.ExecutionStatusFailed:
		return workflow.StatusFailed
	case types.ExecutionStatusTimedOut:
		return workflow.StatusTimedOut
	case types.ExecutionStatusAborted:
		return workflow.StatusCancelled
	default:
		return workflow.StatusPending
	}
}

func (e *Engine) ListExecutions(ctx context.Context, opts workflow.ListOptions) (*workflow.ListResult, error) {
	input := &sfn.ListExecutionsInput{}

	if opts.WorkflowID != "" {
		input.StateMachineArn = aws.String(opts.WorkflowID)
	}
	if opts.Limit > 0 {
		input.MaxResults = int32(opts.Limit)
	}
	if opts.PageToken != "" {
		input.NextToken = aws.String(opts.PageToken)
	}

	output, err := e.client.ListExecutions(ctx, input)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list executions", err)
	}

	result := &workflow.ListResult{
		Executions: make([]*workflow.Execution, len(output.Executions)),
	}

	for i, exec := range output.Executions {
		result.Executions[i] = &workflow.Execution{
			ID:         *exec.ExecutionArn,
			WorkflowID: *exec.StateMachineArn,
			Status:     mapStatus(exec.Status),
			StartedAt:  *exec.StartDate,
		}
		if exec.StopDate != nil {
			result.Executions[i].CompletedAt = *exec.StopDate
		}
	}

	if output.NextToken != nil {
		result.NextPageToken = *output.NextToken
	}

	return result, nil
}

func (e *Engine) Cancel(ctx context.Context, executionID string) error {
	_, err := e.client.StopExecution(ctx, &sfn.StopExecutionInput{
		ExecutionArn: aws.String(executionID),
	})
	if err != nil {
		return pkgerrors.Internal("failed to cancel execution", err)
	}
	return nil
}

// CallbackSignal is the payload shape for Signal using the waitForTaskToken pattern.
//
// Step Functions has no Temporal-style named signals. Callers pass signalName
// "task_success" or "task_failure" (or any name) with CallbackSignal / map data
// containing TaskToken; this adapter calls SendTaskSuccess / SendTaskFailure.
type CallbackSignal struct {
	// TaskToken is the waitForTaskToken callback token from the activity context.
	TaskToken string `json:"task_token"`

	// Output is JSON-serializable success payload (task_success).
	Output interface{} `json:"output,omitempty"`

	// Error / Cause are used for task_failure.
	Error string `json:"error,omitempty"`
	Cause string `json:"cause,omitempty"`
}

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	_ = executionID // SFN callback tokens identify the task, not the execution ARN
	cb, err := parseCallbackSignal(data)
	if err != nil {
		return err
	}
	if cb.TaskToken == "" {
		return pkgerrors.InvalidArgument("stepfunctions Signal requires task_token (waitForTaskToken callback)", nil)
	}

	switch signalName {
	case "task_failure", "failure", "fail":
		_, err := e.client.SendTaskFailure(ctx, &sfn.SendTaskFailureInput{
			TaskToken: aws.String(cb.TaskToken),
			Error:     aws.String(cb.Error),
			Cause:     aws.String(cb.Cause),
		})
		if err != nil {
			return pkgerrors.Internal("failed to send task failure", err)
		}
		return nil
	default:
		out := "{}"
		if cb.Output != nil {
			raw, err := json.Marshal(cb.Output)
			if err != nil {
				return pkgerrors.InvalidArgument("failed to marshal callback output", err)
			}
			out = string(raw)
		}
		_, err := e.client.SendTaskSuccess(ctx, &sfn.SendTaskSuccessInput{
			TaskToken: aws.String(cb.TaskToken),
			Output:    aws.String(out),
		})
		if err != nil {
			return pkgerrors.Internal("failed to send task success", err)
		}
		return nil
	}
}

func parseCallbackSignal(data interface{}) (CallbackSignal, error) {
	switch v := data.(type) {
	case CallbackSignal:
		return v, nil
	case *CallbackSignal:
		if v == nil {
			return CallbackSignal{}, pkgerrors.InvalidArgument("callback signal is nil", nil)
		}
		return *v, nil
	case string:
		return CallbackSignal{TaskToken: v}, nil
	case map[string]interface{}:
		raw, err := json.Marshal(v)
		if err != nil {
			return CallbackSignal{}, pkgerrors.InvalidArgument("invalid callback signal map", err)
		}
		var cb CallbackSignal
		if err := json.Unmarshal(raw, &cb); err != nil {
			return CallbackSignal{}, pkgerrors.InvalidArgument("invalid callback signal map", err)
		}
		if cb.TaskToken == "" {
			if t, ok := v["taskToken"].(string); ok {
				cb.TaskToken = t
			}
		}
		return cb, nil
	default:
		raw, err := json.Marshal(data)
		if err != nil {
			return CallbackSignal{}, pkgerrors.InvalidArgument("unsupported callback signal type", err)
		}
		var cb CallbackSignal
		if err := json.Unmarshal(raw, &cb); err != nil {
			return CallbackSignal{}, pkgerrors.InvalidArgument("unsupported callback signal type", err)
		}
		return cb, nil
	}
}

func (e *Engine) Wait(ctx context.Context, executionID string) (*workflow.Execution, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			exec, err := e.GetExecution(ctx, executionID)
			if err != nil {
				return nil, err
			}
			if exec.Status != workflow.StatusRunning && exec.Status != workflow.StatusPending {
				return exec, nil
			}
		}
	}
}
