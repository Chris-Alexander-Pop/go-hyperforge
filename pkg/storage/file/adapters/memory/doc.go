// Package memory provides an in-memory implementation of file.FileStore.
//
// Uses pkg/concurrency.SmartRWMutex for locking. Intended for tests and
// local development only — not a production network filesystem.
package memory
