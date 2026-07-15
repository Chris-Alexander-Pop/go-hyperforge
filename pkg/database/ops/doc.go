/*
Package ops provides bulk SQL helpers (upsert, delete, raw exec, transactions)
and WithRetry for transient failures.

WithRetry is a thin wrapper around pkg/resilience.Retry. For circuit breaking
on SQL backends, wrap connections with sql.NewResilientSQL — this package does
not implement a circuit breaker of its own.
*/
package ops
