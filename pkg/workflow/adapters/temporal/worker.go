package temporal

import (
	"fmt"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	sdkworkflow "go.temporal.io/sdk/workflow"

	pkgworkflow "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
)

// WorkflowRegistry is the Temporal worker registration surface used by helpers.
// worker.Worker satisfies this interface.
type WorkflowRegistry interface {
	RegisterWorkflow(w interface{})
	RegisterWorkflowWithOptions(w interface{}, options sdkworkflow.RegisterOptions)
	RegisterActivity(a interface{})
}

// WorkerConfig configures NewWorker.
type WorkerConfig struct {
	// TaskQueue is the Temporal task queue (required).
	TaskQueue string

	// Options are forwarded to worker.New (zero value is fine).
	Options worker.Options
}

// NewWorker creates a Temporal worker and registers the given workflow functions.
// Named entries use RegisterWorkflowWithOptions; unnamed (empty key) use RegisterWorkflow.
// WorkflowDefinition values in the map are skipped (engine metadata only).
func NewWorker(c client.Client, cfg WorkerConfig, namedWorkflows map[string]interface{}) (worker.Worker, error) {
	if c == nil {
		return nil, fmt.Errorf("temporal client is required")
	}
	if cfg.TaskQueue == "" {
		return nil, fmt.Errorf("task queue is required")
	}
	w := worker.New(c, cfg.TaskQueue, cfg.Options)
	RegisterWorkflows(w, namedWorkflows)
	return w, nil
}

// RegisterWorkflows registers workflow funcs onto reg.
func RegisterWorkflows(reg WorkflowRegistry, named map[string]interface{}) {
	if reg == nil || len(named) == 0 {
		return
	}
	for name, fn := range named {
		if fn == nil {
			continue
		}
		if _, ok := fn.(pkgworkflow.WorkflowDefinition); ok {
			continue
		}
		if name == "" {
			reg.RegisterWorkflow(fn)
			continue
		}
		reg.RegisterWorkflowWithOptions(fn, sdkworkflow.RegisterOptions{Name: name})
	}
}

// RegisterActivities registers activity funcs onto reg.
func RegisterActivities(reg WorkflowRegistry, activities ...interface{}) {
	if reg == nil {
		return
	}
	for _, a := range activities {
		if a == nil {
			continue
		}
		reg.RegisterActivity(a)
	}
}

// NewWorkerFromEngine builds a worker on task queue from engine config and
// registers workflow funcs previously stored via RegisterWorkflowType.
// The live Temporal client must be passed separately (Engine may hold a test double).
func (e *Engine) NewWorkerFromEngine(c client.Client, opts worker.Options) (worker.Worker, error) {
	if e == nil {
		return nil, fmt.Errorf("engine is required")
	}
	tq := e.config.TaskQueue
	if tq == "" {
		tq = "default-task-queue"
	}
	return NewWorker(c, WorkerConfig{TaskQueue: tq, Options: opts}, e.workflows)
}
