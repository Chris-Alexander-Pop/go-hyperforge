package tests

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/test"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	workflowmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/memory"
)

// EngineSuite adopts pkg/test.Suite for a focused WorkflowEngine smoke path.
// Broader coverage remains in workflow_test.go (testify suite).
type EngineSuite struct {
	test.Suite
	engine workflow.WorkflowEngine
}

func (s *EngineSuite) SetupTest() {
	s.Suite.SetupTest()
	s.engine = workflowmemory.New()
}

func (s *EngineSuite) TestRegisterGetAndStart() {
	def := workflow.WorkflowDefinition{
		ID:   "suite-order",
		Name: "Suite Order",
		States: []workflow.State{
			{Name: "validate", Type: "Task", Next: "done"},
			{Name: "done", Type: "Task", End: true},
		},
		StartAt: "validate",
	}
	s.NoError(s.engine.RegisterWorkflow(s.Ctx, def))

	got, err := s.engine.GetWorkflow(s.Ctx, "suite-order")
	s.NoError(err)
	s.Equal("Suite Order", got.Name)

	exec, err := s.engine.Start(s.Ctx, workflow.StartOptions{
		WorkflowID: "suite-order",
		Input:      map[string]string{"order": "1"},
	})
	s.NoError(err)
	s.NotEmpty(exec.ID)
	s.Equal("suite-order", exec.WorkflowID)
}

func (s *EngineSuite) TestGetWorkflowNotFound() {
	_, err := s.engine.GetWorkflow(s.Ctx, "missing")
	s.Error(err)
}

func TestEngineSuite(t *testing.T) {
	test.Run(t, new(EngineSuite))
}
