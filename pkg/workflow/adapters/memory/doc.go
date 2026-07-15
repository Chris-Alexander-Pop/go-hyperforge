// Package memory provides an in-memory implementation of workflow.WorkflowEngine.
//
// Uses pkg/concurrency.SmartRWMutex for locking. Honors StartOptions.Timeout
// (falls back to Config.DefaultTimeout) and sets StatusTimedOut on deadline.
package memory
