// Package workflow provides a unified interface for workflow orchestration.
//
// Supported backends:
//   - Memory: In-memory workflow engine for testing
//   - StepFunctions: AWS Step Functions
//   - Temporal: Temporal.io durable execution
//   - LogicApps: Azure Logic Apps
//
// Decorators:
//   - NewInstrumentedWorkflowEngine for logging/tracing
//   - NewEventedEngine for pkg/events lifecycle publishing (start/complete/fail)
//
// Subpackages:
//   - saga: compensating transactions (optional NewInstrumentedSaga)
//   - scheduler: cron jobs with optional Store + distlock.Locker
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/workflow/adapters/memory"
//
//	engine := memory.New()
//	exec, err := engine.Start(ctx, workflow.StartOptions{WorkflowID: "order-123", Input: orderData})
package workflow
