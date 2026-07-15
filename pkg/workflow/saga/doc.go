// Package saga provides the Saga pattern for distributed transactions.
//
// The Saga pattern manages long-running transactions by breaking them into
// a series of local transactions with compensating actions for rollback.
//
// Compensation errors from multiple steps are aggregated (errors.Join) and
// wrapped with pkg/errors.Internal.
//
// Durable execution: use StateStore (adapters/memory or adapters/file) with
// NewDurableExecutor to persist progress and Resume after a crash (at-least-once
// step semantics).
//
// Optional observability: wrap with NewInstrumentedSaga for tracing/logging.
//
// Usage:
//
//	saga := saga.New("order-saga")
//	saga.AddStep(saga.Step{Name: "reserve-inventory", Action: reserveInventory, Compensate: releaseInventory})
//	saga.AddStep(saga.Step{Name: "charge-payment", Action: chargePayment, Compensate: refundPayment})
//	result, err := saga.Execute(ctx, orderData)
package saga
