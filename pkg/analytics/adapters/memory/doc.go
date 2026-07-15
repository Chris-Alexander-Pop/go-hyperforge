/*
Package memory implements an in-memory analytics.Tracker backed by
pkg/datastructures/hyperloglog, plus an Event Sink for local ingest tests.

Suitable for unit tests and single-process uniqueness tracking.
Precision follows analytics.Config (4–16). Call Close when finished.
*/
package memory
