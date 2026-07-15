// Package paypal provides the PayPal payment adapter.
//
// Features:
//   - Charge (capture approved order) / Refund / GetTransaction
//   - Authorize / Capture / Void for auth-capture flows
//   - WebhookVerifier via PayPal VerifyWebhookSignature API
//   - NewLocalWebhookVerifier for offline fixture tests
//   - pkg/resilience retries around SDK calls
//
// Amounts use commerce.Money (int64 minor units).
package paypal
