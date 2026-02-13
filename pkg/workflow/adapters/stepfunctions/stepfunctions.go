// Package stepfunctions provides an AWS Step Functions adapter for workflow.WorkflowEngine.
//
// This adapter wraps the AWS SDK to manage Step Functions state machines and executions.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/workflow/adapters/stepfunctions"
//
//	engine, err := stepfunctions.New(stepfunctions.Config{Region: "us-east-1"})
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
	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/workflow"
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
}

// Engine implements workflow.WorkflowEngine for AWS Step Functions.
type Engine struct {
	client *sfn.Client
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

	return &Engine{
		client: sfn.NewFromConfig(awsCfg, clientOpts...),
		config: cfg,
	}, nil
}

func (e *Engine) RegisterWorkflow(ctx context.Context, def workflow.WorkflowDefinition) error {
	// Step Functions state machines are created via CloudFormation or API
	// This creates a new state machine
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
		RoleArn:    aws.String("arn:aws:iam::123456789012:role/StepFunctionsRole"), // Placeholder
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

func (e *Engine) Signal(ctx context.Context, executionID string, signalName string, data interface{}) error {
	// Step Functions doesn't support signals in the same way as Temporal
	// This would require using a callback pattern with SQS/SNS
	return pkgerrors.Internal("signals not supported for Step Functions", nil)
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
