// Package stepfunctions provides an AWS Step Functions adapter for workflow.WorkflowEngine.
//
// Shipped:
//   - CreateStateMachine (RegisterWorkflow) using Config.RoleArn
//   - Start / Describe / List / Stop executions
//   - Signal via waitForTaskToken callback stub (SendTaskSuccess / SendTaskFailure)
//
// Remaining gaps (honest):
//   - Full ASL conversion from workflow.State (Choice/Parallel/Map) is minimal
//   - Activity worker GetActivityTask loop is not hosted here
//   - Signal does not target an execution ARN; callers must supply the task token
package stepfunctions
