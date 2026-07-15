package temporal_test

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/temporal"
	"go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMapTemporalStatus(t *testing.T) {
	cases := []struct {
		in   enumspb.WorkflowExecutionStatus
		want workflow.ExecutionStatus
	}{
		{enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING, workflow.StatusRunning},
		{enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, workflow.StatusCompleted},
		{enumspb.WORKFLOW_EXECUTION_STATUS_FAILED, workflow.StatusFailed},
		{enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED, workflow.StatusCancelled},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED, workflow.StatusCancelled},
		{enumspb.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW, workflow.StatusRunning},
		{enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT, workflow.StatusTimedOut},
		{enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED, workflow.StatusPending},
		{enumspb.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED, workflow.StatusPending},
	}
	for _, tc := range cases {
		if got := temporal.MapTemporalStatus(tc.in); got != tc.want {
			t.Fatalf("status %v: got %s want %s", tc.in, got, tc.want)
		}
	}
}

type fakeClient struct {
	listReq  *workflowservice.ListWorkflowExecutionsRequest
	listResp *workflowservice.ListWorkflowExecutionsResponse
	closed   bool
}

func (f *fakeClient) ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, args ...interface{}) (client.WorkflowRun, error) {
	return nil, nil
}
func (f *fakeClient) DescribeWorkflowExecution(ctx context.Context, workflowID, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	return &workflowservice.DescribeWorkflowExecutionResponse{
		WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
			Execution: &common.WorkflowExecution{WorkflowId: workflowID, RunId: runID},
			Status:    enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED,
			StartTime: timestamppb.New(time.Unix(100, 0)),
			CloseTime: timestamppb.New(time.Unix(200, 0)),
		},
	}, nil
}
func (f *fakeClient) ListWorkflow(ctx context.Context, request *workflowservice.ListWorkflowExecutionsRequest) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	f.listReq = request
	if f.listResp != nil {
		return f.listResp, nil
	}
	return &workflowservice.ListWorkflowExecutionsResponse{}, nil
}
func (f *fakeClient) CancelWorkflow(ctx context.Context, workflowID, runID string) error { return nil }
func (f *fakeClient) SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg interface{}) error {
	return nil
}
func (f *fakeClient) GetWorkflow(ctx context.Context, workflowID, runID string) client.WorkflowRun {
	return nil
}
func (f *fakeClient) Close() { f.closed = true }

func TestListExecutionsVisibilityQuery(t *testing.T) {
	fc := &fakeClient{
		listResp: &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflowpb.WorkflowExecutionInfo{
				{
					Execution: &common.WorkflowExecution{WorkflowId: "wf-1", RunId: "run-1"},
					Status:    enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING,
					StartTime: timestamppb.Now(),
				},
			},
			NextPageToken: []byte("next"),
		},
	}
	eng := temporal.NewFromClient(fc, temporal.Config{Namespace: "prod"}, false)
	defer eng.Close()

	res, err := eng.ListExecutions(context.Background(), workflow.ListOptions{
		WorkflowID: "wf-1",
		Status:     workflow.StatusRunning,
		Limit:      10,
		PageToken:  base64.StdEncoding.EncodeToString([]byte("tok")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if fc.listReq == nil || fc.listReq.PageSize != 10 {
		t.Fatalf("unexpected list req: %+v", fc.listReq)
	}
	wantQ := `WorkflowId = "wf-1" AND ExecutionStatus = "Running"`
	if fc.listReq.Query != wantQ {
		t.Fatalf("query=%q want %q", fc.listReq.Query, wantQ)
	}
	if len(res.Executions) != 1 || res.Executions[0].ID != "run-1" {
		t.Fatalf("executions=%+v", res.Executions)
	}
	if res.NextPageToken != base64.StdEncoding.EncodeToString([]byte("next")) {
		t.Fatalf("next token=%q", res.NextPageToken)
	}
}

func TestGetExecutionAndClose(t *testing.T) {
	fc := &fakeClient{}
	eng := temporal.NewFromClient(fc, temporal.Config{}, true)
	exec, err := eng.GetExecution(context.Background(), "wf/run-abc")
	if err != nil {
		t.Fatal(err)
	}
	if exec.WorkflowID != "wf" || exec.ID != "run-abc" || exec.Status != workflow.StatusCompleted {
		t.Fatalf("exec=%+v", exec)
	}
	eng.Close()
	if !fc.closed {
		t.Fatal("expected client Close")
	}
}
