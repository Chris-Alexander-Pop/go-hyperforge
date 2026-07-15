package temporal_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/temporal"
	"go.temporal.io/sdk/worker"
	sdkworkflow "go.temporal.io/sdk/workflow"
)

type fakeRegistry struct {
	workflows  []string
	activities int
}

func (f *fakeRegistry) RegisterWorkflow(w interface{}) {
	f.workflows = append(f.workflows, "")
}

func (f *fakeRegistry) RegisterWorkflowWithOptions(w interface{}, options sdkworkflow.RegisterOptions) {
	f.workflows = append(f.workflows, options.Name)
}

func (f *fakeRegistry) RegisterActivity(a interface{}) {
	f.activities++
}

func sampleWF(ctx sdkworkflow.Context) error { return nil }

func TestRegisterWorkflowsSkipsDefinitions(t *testing.T) {
	reg := &fakeRegistry{}
	temporal.RegisterWorkflows(reg, map[string]interface{}{
		"Sample": sampleWF,
		"meta":   workflow.WorkflowDefinition{ID: "meta"},
		"":       sampleWF,
	})
	if len(reg.workflows) != 2 {
		t.Fatalf("registered=%v want 2", reg.workflows)
	}
	temporal.RegisterActivities(reg, sampleWF, nil)
	if reg.activities != 1 {
		t.Fatalf("activities=%d", reg.activities)
	}
}

func TestNewWorkerRequiresClientAndQueue(t *testing.T) {
	_, err := temporal.NewWorker(nil, temporal.WorkerConfig{TaskQueue: "q"}, nil)
	if err == nil {
		t.Fatal("expected error for nil client")
	}
	// Cannot dial without server; NewWorkerFromEngine still validates via NewWorker.
	eng := temporal.NewFromClient(&fakeClient{}, temporal.Config{TaskQueue: "tq"}, false)
	_, err = eng.NewWorkerFromEngine(nil, worker.Options{})
	if err == nil {
		t.Fatal("expected error for nil client on NewWorkerFromEngine")
	}
}
