// Package temporal provides a Temporal.io adapter for workflow.WorkflowEngine.
//
// Shipped:
//   - Start / GetExecution / Cancel / Signal / Wait via go.temporal.io/sdk/client
//   - Status mapping via enums.WorkflowExecutionStatus (MapTemporalStatus)
//   - ListExecutions via visibility ListWorkflow with WorkflowId/ExecutionStatus query
//   - Close() releases the dialed client (NewFromClient can opt out)
//
// Remaining gaps (honest):
//   - Worker registration / activity hosting is out of scope for this engine wrapper
//   - Advanced visibility (custom SearchAttributes, CountWorkflow) not exposed
//   - Signal/Query typed helpers beyond raw SignalWorkflow
package temporal
