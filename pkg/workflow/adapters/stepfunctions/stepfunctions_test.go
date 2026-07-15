package stepfunctions_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/stepfunctions"
)

type fakeSFN struct {
	roleArn       string
	successToken  string
	successOutput string
	failToken     string
}

func (f *fakeSFN) CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, _ ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
	f.roleArn = aws.ToString(params.RoleArn)
	return &sfn.CreateStateMachineOutput{StateMachineArn: aws.String("arn:sm")}, nil
}
func (f *fakeSFN) DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, _ ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	return nil, nil
}
func (f *fakeSFN) StartExecution(ctx context.Context, params *sfn.StartExecutionInput, _ ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	return nil, nil
}
func (f *fakeSFN) DescribeExecution(ctx context.Context, params *sfn.DescribeExecutionInput, _ ...func(*sfn.Options)) (*sfn.DescribeExecutionOutput, error) {
	return &sfn.DescribeExecutionOutput{
		ExecutionArn:    params.ExecutionArn,
		StateMachineArn: aws.String("arn:sm"),
		Status:          types.ExecutionStatusSucceeded,
		StartDate:       aws.Time(time.Unix(1, 0)),
		StopDate:        aws.Time(time.Unix(2, 0)),
	}, nil
}
func (f *fakeSFN) ListExecutions(ctx context.Context, params *sfn.ListExecutionsInput, _ ...func(*sfn.Options)) (*sfn.ListExecutionsOutput, error) {
	return &sfn.ListExecutionsOutput{}, nil
}
func (f *fakeSFN) StopExecution(ctx context.Context, params *sfn.StopExecutionInput, _ ...func(*sfn.Options)) (*sfn.StopExecutionOutput, error) {
	return &sfn.StopExecutionOutput{}, nil
}
func (f *fakeSFN) SendTaskSuccess(ctx context.Context, params *sfn.SendTaskSuccessInput, _ ...func(*sfn.Options)) (*sfn.SendTaskSuccessOutput, error) {
	f.successToken = aws.ToString(params.TaskToken)
	f.successOutput = aws.ToString(params.Output)
	return &sfn.SendTaskSuccessOutput{}, nil
}
func (f *fakeSFN) SendTaskFailure(ctx context.Context, params *sfn.SendTaskFailureInput, _ ...func(*sfn.Options)) (*sfn.SendTaskFailureOutput, error) {
	f.failToken = aws.ToString(params.TaskToken)
	return &sfn.SendTaskFailureOutput{}, nil
}

func TestRoleArnFromConfig(t *testing.T) {
	api := &fakeSFN{}
	eng := stepfunctions.NewFromAPI(api, stepfunctions.Config{RoleArn: "arn:aws:iam::1:role/sfn"})
	err := eng.RegisterWorkflow(context.Background(), workflow.WorkflowDefinition{
		Name: "demo", StartAt: "A", States: []workflow.State{{Name: "A", Type: "Pass", End: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.roleArn != "arn:aws:iam::1:role/sfn" {
		t.Fatalf("roleArn=%q", api.roleArn)
	}
}

func TestRegisterRequiresRoleArn(t *testing.T) {
	eng := stepfunctions.NewFromAPI(&fakeSFN{}, stepfunctions.Config{})
	err := eng.RegisterWorkflow(context.Background(), workflow.WorkflowDefinition{Name: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSignalCallbackToken(t *testing.T) {
	api := &fakeSFN{}
	eng := stepfunctions.NewFromAPI(api, stepfunctions.Config{})
	err := eng.Signal(context.Background(), "arn:exec", "task_success", stepfunctions.CallbackSignal{
		TaskToken: "tok-1",
		Output:    map[string]string{"ok": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.successToken != "tok-1" || api.successOutput == "" {
		t.Fatalf("success token=%q out=%q", api.successToken, api.successOutput)
	}
	err = eng.Signal(context.Background(), "arn:exec", "task_failure", map[string]interface{}{
		"task_token": "tok-2", "error": "Boom",
	})
	if err != nil {
		t.Fatal(err)
	}
	if api.failToken != "tok-2" {
		t.Fatalf("fail token=%q", api.failToken)
	}
}
