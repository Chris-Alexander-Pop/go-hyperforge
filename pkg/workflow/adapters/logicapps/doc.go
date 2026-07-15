// Package logicapps provides an Azure Logic Apps adapter for workflow.WorkflowEngine.
//
// Shipped:
//   - Trigger manual runs; capture run id from response headers when present
//   - GetExecution fetches remote run status via ARM workflows/.../runs/{id}
//   - ListExecutions from ARM run history; Cancel via cancel action when known
//   - Close() clears local caches
//
// Remaining gaps (honest):
//   - Full ARM template deployment for RegisterWorkflow is local-only
//   - Native Signal/callback is Unimplemented (use HTTP webhook actions in the app)
//   - Token refresh / MSI auth paths are not implemented (client-credentials only)
package logicapps
