// Package eventsource provides Event Sourcing patterns.
//
// Event versions are 1-based. LoadFrom(fromVersion) returns events whose
// Version is greater than or equal to fromVersion (not a slice index).
//
// For local fan-out after Append, wrap an EventStore with EventedStore and a
// pkg/events.Bus. For distributed delivery, publish from a projection onto
// pkg/messaging instead.
package eventsource
