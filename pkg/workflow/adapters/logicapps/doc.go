// Package logicapps provides an Azure Logic Apps adapter for workflow.WorkflowEngine.
//
// Shipped:
//   - Trigger manual runs; capture run id from response headers when present
//   - GetExecution fetches remote run status via ARM workflows/.../runs/{id}
//   - ListExecutions from ARM run history; Cancel via cancel action when known
//   - Close() clears local caches; concurrency.SmartRWMutex for local maps
//   - ARM auth modes: client_secret (AAD), managed_identity (IMDS; IdentityBase injectable),
//     default (azidentity.DefaultAzureCredential), optional TokenSource override
//   - SkipAuth + HTTPClient/ManagementBase/LoginBase/IdentityBase for httptest unit tests
//
// Remaining gaps (honest):
//   - Full ARM template deployment for RegisterWorkflow is local-only
//   - Native Signal/callback is Unimplemented (use HTTP webhook actions in the app)
//   - Tokens are fetched once at New; no background refresh before expiry
package logicapps
