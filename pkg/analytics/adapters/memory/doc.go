/*
Package memory implements an in-memory analytics.Tracker backed by
pkg/datastructures/hyperloglog.

Suitable for unit tests and single-process uniqueness tracking.
Precision follows analytics.Config (4–16). Call Close when finished.
*/
package memory
