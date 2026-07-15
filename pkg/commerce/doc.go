/*
Package commerce provides e-commerce and payment processing capabilities.

Shared types (Money) live in this root package. Subpackages:

  - billing: Subscription and invoice management
  - currency: Currency conversion and formatting
  - payment: Payment gateway integrations (Stripe, PayPal; Braintree not included)
  - tax: Tax calculation services

Money uses int64 minor units — never float64 for payment amounts.

Usage:

	import (
		"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
		"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/payment"
	)

	amount := commerce.NewMoney(1000, "USD") // $10.00
	result, err := processor.Charge(ctx, &payment.ChargeRequest{Amount: amount, SourceID: "tok_visa"})
*/
package commerce
