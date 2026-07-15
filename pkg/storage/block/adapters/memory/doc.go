// Package memory provides an in-memory implementation of block.VolumeStore.
//
// Uses pkg/concurrency.SmartRWMutex for locking. Intended for tests and
// local development only — not a production block storage backend.
package memory
