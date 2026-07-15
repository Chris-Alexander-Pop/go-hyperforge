package memory_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/memory"
)

func TestChoiceAndParallel(t *testing.T) {
	eng := memory.New().(*memory.Engine)
	ctx := context.Background()

	var aCalls, bCalls atomic.Int32
	eng.RegisterTaskHandler("pathA", func(ctx context.Context, input interface{}) (interface{}, error) {
		aCalls.Add(1)
		return map[string]interface{}{"path": "A"}, nil
	})
	eng.RegisterTaskHandler("pathB", func(ctx context.Context, input interface{}) (interface{}, error) {
		bCalls.Add(1)
		return map[string]interface{}{"path": "B"}, nil
	})
	eng.RegisterTaskHandler("left", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "L", nil
	})
	eng.RegisterTaskHandler("right", func(ctx context.Context, input interface{}) (interface{}, error) {
		return "R", nil
	})

	if err := eng.RegisterWorkflow(ctx, workflow.WorkflowDefinition{
		ID:      "choice-wf",
		StartAt: "choose",
		States: []workflow.State{
			{
				Name: "choose",
				Type: "Choice",
				Choices: []workflow.ChoiceRule{
					{Variable: "$.route", StringEquals: "a", Next: "doA"},
				},
				Default: "doB",
			},
			{Name: "doA", Type: "Task", Resource: "pathA", End: true},
			{Name: "doB", Type: "Task", Resource: "pathB", End: true},
		},
	}); err != nil {
		t.Fatal(err)
	}

	exec, err := eng.Start(ctx, workflow.StartOptions{
		WorkflowID: "choice-wf",
		Input:      map[string]interface{}{"route": "a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	done, err := eng.Wait(ctx, exec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != workflow.StatusCompleted {
		t.Fatalf("status=%s err=%s", done.Status, done.Error)
	}
	out := done.Output.(map[string]interface{})
	if out["path"] != "A" || aCalls.Load() != 1 || bCalls.Load() != 0 {
		t.Fatalf("choice A failed: out=%v a=%d b=%d", out, aCalls.Load(), bCalls.Load())
	}

	if err := eng.RegisterWorkflow(ctx, workflow.WorkflowDefinition{
		ID:      "parallel-wf",
		StartAt: "fan",
		States: []workflow.State{
			{
				Name: "fan",
				Type: "Parallel",
				Branches: []workflow.Branch{
					{StartAt: "left", States: []workflow.State{
						{Name: "left", Type: "Task", Resource: "left", End: true},
					}},
					{StartAt: "right", States: []workflow.State{
						{Name: "right", Type: "Task", Resource: "right", End: true},
					}},
				},
				End: true,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	exec2, err := eng.Start(ctx, workflow.StartOptions{WorkflowID: "parallel-wf", Input: "in"})
	if err != nil {
		t.Fatal(err)
	}
	wctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	done2, err := eng.Wait(wctx, exec2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done2.Status != workflow.StatusCompleted {
		t.Fatalf("parallel status=%s err=%s", done2.Status, done2.Error)
	}
	arr, ok := done2.Output.([]interface{})
	if !ok || len(arr) != 2 || arr[0] != "L" || arr[1] != "R" {
		t.Fatalf("parallel output=%v", done2.Output)
	}
}
