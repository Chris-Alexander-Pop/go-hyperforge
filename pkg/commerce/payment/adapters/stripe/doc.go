// Package stripe provides the Stripe payment adapter.
//
// Features:
//   - Charge / Refund / GetTransaction
//   - Authorize / Capture / Void (manual capture PaymentIntents)
//   - WebhookVerifier via Stripe signing secret (webhook.ConstructEvent)
//   - pkg/resilience retries around SDK calls
//
// Amounts use commerce.Money (int64 minor units). IdempotencyKey is forwarded
// to Stripe's Idempotency-Key header.
package stripe
