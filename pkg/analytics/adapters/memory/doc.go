/*
Package memory implements an in-memory analytics.Tracker backed by
pkg/datastructures/hyperloglog, an Event Sink for local ingest tests, and
ExactStore implementing analytics.CounterStore (non-HLL exact counts / sets).

Suitable for unit tests and single-process uniqueness tracking.
Precision follows analytics.Config (4–16) for HLL Tracker. Call Close when finished.
*/
package memory
