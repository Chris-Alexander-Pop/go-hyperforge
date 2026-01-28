/*
Package commerce provides e-commerce and payment processing capabilities.

Subpackages:

  - billing: Subscription and invoice management
  - currency: Currency conversion and formatting
  - payment: Payment gateway integrations (Stripe, Braintree, PayPal)
  - tax: Tax calculation services

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"

	processor, err := stripe.New(cfg)
	result, err := processor.Charge(ctx, payment.ChargeRequest{Amount: 1000})
*/
package commerce
