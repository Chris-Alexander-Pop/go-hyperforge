/*
Package memory provides in-memory EventStore and SnapshotStore adapters.

Uses pkg/concurrency.SmartRWMutex for observability-friendly locking.
Suitable for unit tests and single-process development.
*/
package memory
