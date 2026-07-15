// Package eventsource provides Event Sourcing patterns.
//
// Event versions are 1-based. LoadFrom(fromVersion) returns events whose
// Version is greater than or equal to fromVersion (not a slice index).
//
// For local fan-out after Append, wrap an EventStore with EventedStore and a
// pkg/events.Bus. For distributed delivery to projections/integrations, use
// NewEventedStoreWithOutbox (local bus + pkg/messaging outbox).
//
// ProjectionRunner catch-up-projects LoadAll events onto a cqrs.Projector with
// a durable CheckpointStore (memory or adapters/sql / adapters/postgres).
// Use Run for continuous catch-up with error backoff; ResetCheckpoint to
// restart from zero; NewInstrumentedProjectionRunner / ProjectionMetrics for
// observability; Config + NewProjectionFromConfig for env-driven defaults.
package eventsource
