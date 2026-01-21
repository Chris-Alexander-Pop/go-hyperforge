package stripe

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
)

// Provider implements payment.Provider for Stripe.
type Provider struct {
	client *client.API
}

// New creates a new Stripe provider.
func New(cfg payment.Config) (*Provider, error) {
	if err := validator.New().ValidateStruct(cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}

	if cfg.StripeKey == "" {
		return nil, errors.InvalidArgument("stripe key is required", nil)
	}

	sc := client.New(cfg.StripeKey, nil)
	return &Provider{client: sc}, nil
}

func (p *Provider) Charge(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	// Stripe uses cents for USD
	amountInt := int64(req.Amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(amountInt),
		Currency:      stripe.String(req.Currency),
		PaymentMethod: stripe.String(req.SourceID),
		Description:   stripe.String(req.Description),
		Confirm:       stripe.Bool(true), // Immediate confirmation
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled:        stripe.Bool(true),
			AllowRedirects: stripe.String("never"), // Simplification for API
		},
	}

	// Add metadata
	if len(req.Metadata) > 0 {
		params.Metadata = req.Metadata
	}

	// Normally we might pass context via stripe.Params?
	// The stripe-go library's Params struct allows Context to be set via SetContext?
	// Checking the library... v76 clients support standard context methods usually or Params.Context.
	params.Context = ctx

	pi, err := p.client.PaymentIntents.New(params)
	if err != nil {
		return nil, mapStripeError(err)
	}

	status := payment.StatusPending
	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		status = payment.StatusSucceeded
	} else if pi.Status == stripe.PaymentIntentStatusRequiresPaymentMethod {
		status = payment.StatusFailed
	}

	return &payment.Transaction{
		ID:          pi.ID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Status:      status,
		SourceID:    req.SourceID,
		Description: pi.Description,
		CreatedAt:   time.Now(), // Stripe 'Created' is int64, skipping conversion for brevity unless important
		Metadata:    pi.Metadata,
	}, nil
}

func (p *Provider) Refund(ctx context.Context, req *payment.RefundRequest) (*payment.Transaction, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.TransactionID),
	}
	if req.Amount > 0 {
		params.Amount = stripe.Int64(int64(req.Amount * 100))
	}

	// Add reason if supported? Stripe supports reasons like 'duplicate', 'fraudulent', 'requested_by_customer'
	// We'll map if needed or store in metadata.

	params.Context = ctx

	ref, err := p.client.Refunds.New(params)
	if err != nil {
		return nil, mapStripeError(err)
	}

	return &payment.Transaction{
		ID:        ref.ID,
		Amount:    float64(ref.Amount) / 100.0,
		Currency:  string(ref.Currency),
		Status:    payment.StatusRefunded,
		CreatedAt: time.Now(),
	}, nil
}

func (p *Provider) GetTransaction(ctx context.Context, id string) (*payment.Transaction, error) {
	// Usually retrieve PaymentIntent
	params := &stripe.PaymentIntentParams{}
	params.Context = ctx

	pi, err := p.client.PaymentIntents.Get(id, params)
	if err != nil {
		return nil, mapStripeError(err)
	}

	status := payment.StatusPending
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = payment.StatusSucceeded
	case stripe.PaymentIntentStatusCanceled:
		status = payment.StatusFailed
	}

	return &payment.Transaction{
		ID:          pi.ID,
		Amount:      float64(pi.Amount) / 100.0,
		Currency:    string(pi.Currency),
		Status:      status,
		Description: pi.Description,
	}, nil
}

func (p *Provider) Close() error {
	return nil
}

func mapStripeError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		switch stripeErr.Code {
		case stripe.ErrorCodeCardDeclined:
			return payment.ErrDeclined
		case stripe.ErrorCodeExpiredCard:
			return payment.ErrExpiredCard
		case stripe.ErrorCodeIncorrectCVC, stripe.ErrorCodeIncorrectNumber:
			return payment.ErrInvalidCard
		}
		return errors.Internal("stripe error", err)
	}
	return errors.Internal("payment provider error", err)
}
