// Package memory provides an in-memory implementation of archive.ArchiveStore.
//
// Uses pkg/concurrency.SmartRWMutex for locking. Intended for tests and
// local development only — not a production cold-storage backend.
package memory
