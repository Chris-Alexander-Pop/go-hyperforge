package stripe

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/payment"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/validator"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
	"github.com/stripe/stripe-go/v76/webhook"
)

// Ensure compile-time interface compliance.
var (
	_ payment.Provider        = (*Provider)(nil)
	_ payment.Authorizer      = (*Provider)(nil)
	_ payment.WebhookVerifier = (*WebhookVerifier)(nil)
)

// Provider implements payment.Authorizer for Stripe.
type Provider struct {
	client   *client.API
	retryCfg resilience.RetryConfig
}

// New creates a new Stripe provider with pkg/resilience retries on SDK calls.
func New(cfg payment.Config) (payment.Provider, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}
	if cfg.StripeKey == "" {
		return nil, errors.InvalidArgument("stripe key is required", nil)
	}

	sc := client.New(cfg.StripeKey, nil)
	p := &Provider{client: sc}
	if cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		p.retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     10 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf:        shouldRetryStripe,
		}
	}
	return p, nil
}

func shouldRetryStripe(err error) bool {
	if err == nil {
		return false
	}
	switch err {
	case payment.ErrDeclined, payment.ErrInsufficientFunds, payment.ErrInvalidCard, payment.ErrExpiredCard:
		return false
	}
	if _, ok := err.(*stripe.Error); ok {
		// Card / request errors are not transient.
		se := err.(*stripe.Error)
		switch se.Type {
		case stripe.ErrorTypeCard, stripe.ErrorTypeInvalidRequest:
			return false
		}
	}
	return true
}

func (p *Provider) withRetry(ctx context.Context, fn func(context.Context) error) error {
	if p.retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, p.retryCfg, fn)
	}
	return fn(ctx)
}

func (p *Provider) Charge(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	if err := req.Amount.Validate(); err != nil {
		return nil, err
	}

	var pi *stripe.PaymentIntent
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.PaymentIntentParams{
			Amount:        stripe.Int64(req.Amount.Amount),
			Currency:      stripe.String(req.Amount.Currency),
			PaymentMethod: stripe.String(req.SourceID),
			Description:   stripe.String(req.Description),
			Confirm:       stripe.Bool(true),
			AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
				Enabled:        stripe.Bool(true),
				AllowRedirects: stripe.String("never"),
			},
		}
		if req.IdempotencyKey != "" {
			params.SetIdempotencyKey(req.IdempotencyKey)
		}
		if len(req.Metadata) > 0 {
			params.Metadata = req.Metadata
		}
		params.Context = ctx
		var e error
		pi, e = p.client.PaymentIntents.New(params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	status := payment.StatusPending
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = payment.StatusSucceeded
	case stripe.PaymentIntentStatusRequiresPaymentMethod:
		status = payment.StatusFailed
	case stripe.PaymentIntentStatusRequiresCapture:
		status = payment.StatusAuthorized
	}

	return &payment.Transaction{
		ID:             pi.ID,
		Amount:         commerce.NewMoney(pi.Amount, string(pi.Currency)),
		Status:         status,
		SourceID:       req.SourceID,
		Description:    pi.Description,
		CreatedAt:      time.Now().UTC(),
		Metadata:       pi.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}

func (p *Provider) Authorize(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	if err := req.Amount.Validate(); err != nil {
		return nil, err
	}

	var pi *stripe.PaymentIntent
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.PaymentIntentParams{
			Amount:        stripe.Int64(req.Amount.Amount),
			Currency:      stripe.String(req.Amount.Currency),
			PaymentMethod: stripe.String(req.SourceID),
			Description:   stripe.String(req.Description),
			Confirm:       stripe.Bool(true),
			CaptureMethod: stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
			AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
				Enabled:        stripe.Bool(true),
				AllowRedirects: stripe.String("never"),
			},
		}
		if req.IdempotencyKey != "" {
			params.SetIdempotencyKey(req.IdempotencyKey)
		}
		if len(req.Metadata) > 0 {
			params.Metadata = req.Metadata
		}
		params.Context = ctx
		var e error
		pi, e = p.client.PaymentIntents.New(params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	status := payment.StatusAuthorized
	if pi.Status == stripe.PaymentIntentStatusSucceeded {
		status = payment.StatusSucceeded
	} else if pi.Status == stripe.PaymentIntentStatusRequiresPaymentMethod {
		status = payment.StatusFailed
	}

	return &payment.Transaction{
		ID:             pi.ID,
		Amount:         commerce.NewMoney(pi.Amount, string(pi.Currency)),
		Status:         status,
		SourceID:       req.SourceID,
		Description:    pi.Description,
		CreatedAt:      time.Now().UTC(),
		Metadata:       pi.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}

func (p *Provider) Capture(ctx context.Context, req *payment.CaptureRequest) (*payment.Transaction, error) {
	var pi *stripe.PaymentIntent
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.PaymentIntentCaptureParams{}
		if !req.Amount.IsZero() {
			params.AmountToCapture = stripe.Int64(req.Amount.Amount)
		}
		params.Context = ctx
		var e error
		pi, e = p.client.PaymentIntents.Capture(req.TransactionID, params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payment.Transaction{
		ID:        pi.ID,
		Amount:    commerce.NewMoney(pi.AmountReceived, string(pi.Currency)),
		Status:    payment.StatusSucceeded,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) Void(ctx context.Context, transactionID string) (*payment.Transaction, error) {
	var pi *stripe.PaymentIntent
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.PaymentIntentCancelParams{}
		params.Context = ctx
		var e error
		pi, e = p.client.PaymentIntents.Cancel(transactionID, params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payment.Transaction{
		ID:        pi.ID,
		Amount:    commerce.NewMoney(pi.Amount, string(pi.Currency)),
		Status:    payment.StatusVoided,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) Refund(ctx context.Context, req *payment.RefundRequest) (*payment.Transaction, error) {
	var ref *stripe.Refund
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.RefundParams{
			PaymentIntent: stripe.String(req.TransactionID),
		}
		if !req.Amount.IsZero() {
			params.Amount = stripe.Int64(req.Amount.Amount)
		}
		params.Context = ctx
		var e error
		ref, e = p.client.Refunds.New(params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payment.Transaction{
		ID:        ref.ID,
		Amount:    commerce.NewMoney(ref.Amount, string(ref.Currency)),
		Status:    payment.StatusRefunded,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) GetTransaction(ctx context.Context, id string) (*payment.Transaction, error) {
	var pi *stripe.PaymentIntent
	err := p.withRetry(ctx, func(ctx context.Context) error {
		params := &stripe.PaymentIntentParams{}
		params.Context = ctx
		var e error
		pi, e = p.client.PaymentIntents.Get(id, params)
		if e != nil {
			return mapStripeError(e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	status := payment.StatusPending
	switch pi.Status {
	case stripe.PaymentIntentStatusSucceeded:
		status = payment.StatusSucceeded
	case stripe.PaymentIntentStatusCanceled:
		status = payment.StatusVoided
	case stripe.PaymentIntentStatusRequiresCapture:
		status = payment.StatusAuthorized
	}

	return &payment.Transaction{
		ID:          pi.ID,
		Amount:      commerce.NewMoney(pi.Amount, string(pi.Currency)),
		Status:      status,
		Description: pi.Description,
		CreatedAt:   time.Unix(pi.Created, 0).UTC(),
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

// WebhookVerifier verifies Stripe webhook signatures using the signing secret.
type WebhookVerifier struct {
	secret string
}

// NewWebhookVerifier creates a Stripe webhook verifier.
func NewWebhookVerifier(webhookSecret string) (*WebhookVerifier, error) {
	if webhookSecret == "" {
		return nil, errors.InvalidArgument("stripe webhook secret is required", nil)
	}
	return &WebhookVerifier{secret: webhookSecret}, nil
}

// Verify validates the Stripe-Signature header and returns a normalized event.
func (v *WebhookVerifier) Verify(ctx context.Context, payload []byte, headers map[string]string) (*payment.WebhookEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sig := headers["Stripe-Signature"]
	if sig == "" {
		sig = headers["stripe-signature"]
	}
	if sig == "" {
		return nil, payment.ErrInvalidWebhook
	}

	evt, err := webhook.ConstructEvent(payload, sig, v.secret)
	if err != nil {
		return nil, errors.New("INVALID_WEBHOOK", "invalid webhook signature", err)
	}

	data := map[string]string{
		"type": string(evt.Type),
		"id":   evt.ID,
	}
	if len(evt.Data.Raw) > 0 {
		var obj map[string]interface{}
		if json.Unmarshal(evt.Data.Raw, &obj) == nil {
			if id, ok := obj["id"].(string); ok {
				data["object_id"] = id
			}
		}
	}

	return &payment.WebhookEvent{
		ID:        evt.ID,
		Type:      string(evt.Type),
		Provider:  "stripe",
		CreatedAt: time.Unix(evt.Created, 0).UTC(),
		Raw:       append([]byte(nil), payload...),
		Data:      data,
	}, nil
}
