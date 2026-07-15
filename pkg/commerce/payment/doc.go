// Package payment provides a unified interface for payment processing.
//
// Supported backends:
//   - Memory: In-memory mock for testing (implements Authorizer).
//   - Stripe: Stripe PaymentIntents + webhook signature verification.
//   - PayPal: PayPal Orders capture + remote webhook verification.
//
// Braintree is not currently shipped as an adapter; use Stripe or PayPal
// (or the memory provider in tests). Amounts use commerce.Money (int64 minor units).
//
// Optional decorators:
//   - NewInstrumentedProvider for logging/tracing
//   - NewEventedProvider for pkg/events domain events
//   - NewResilientProvider for retry/circuit-breaker around any Provider
package payment
