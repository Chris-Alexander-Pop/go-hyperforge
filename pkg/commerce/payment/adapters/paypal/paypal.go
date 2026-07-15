package paypal

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"github.com/chris-alexander-pop/system-design-library/pkg/resilience"
	"github.com/chris-alexander-pop/system-design-library/pkg/validator"
	paypalSDK "github.com/plutov/paypal/v4"
)

// Ensure compile-time interface compliance.
var (
	_ payment.Provider        = (*Provider)(nil)
	_ payment.Authorizer      = (*Provider)(nil)
	_ payment.WebhookVerifier = (*WebhookVerifier)(nil)
)

// Provider implements payment.Authorizer for PayPal.
type Provider struct {
	client   *paypalSDK.Client
	retryCfg resilience.RetryConfig
}

// New creates a new PayPal provider with pkg/resilience retries on SDK calls.
func New(cfg payment.Config) (payment.Provider, error) {
	if err := validator.New().ValidateStruct(context.Background(), cfg); err != nil {
		return nil, errors.InvalidArgument("invalid config", err)
	}
	if cfg.PayPalClientID == "" || cfg.PayPalSecret == "" {
		return nil, errors.InvalidArgument("paypal credentials required", nil)
	}

	base := paypalSDK.APIBaseLive
	if cfg.PayPalSandbox {
		base = paypalSDK.APIBaseSandBox
	}

	c, err := paypalSDK.NewClient(cfg.PayPalClientID, cfg.PayPalSecret, base)
	if err != nil {
		return nil, errors.Internal("failed to create paypal client", err)
	}
	_, err = c.GetAccessToken(context.Background())
	if err != nil {
		return nil, errors.Internal("failed to authenticate with paypal", err)
	}

	p := &Provider{client: c}
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
			RetryIf: func(err error) bool {
				if err == nil {
					return false
				}
				switch err {
				case payment.ErrDeclined, payment.ErrInsufficientFunds, payment.ErrInvalidCard:
					return false
				}
				return true
			},
		}
	}
	return p, nil
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

	// Charge captures an approved Order identified by SourceID.
	var capture *paypalSDK.CaptureOrderResponse
	err := p.withRetry(ctx, func(ctx context.Context) error {
		var e error
		capture, e = p.client.CaptureOrder(ctx, req.SourceID, paypalSDK.CaptureOrderRequest{})
		if e != nil {
			return errors.Internal("failed to capture paypal order", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if capture.Status != "COMPLETED" {
		return nil, payment.ErrDeclined
	}

	return &payment.Transaction{
		ID:             capture.ID,
		Amount:         req.Amount,
		Status:         payment.StatusSucceeded,
		SourceID:       req.SourceID,
		Description:    req.Description,
		CreatedAt:      time.Now().UTC(),
		Metadata:       req.Metadata,
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}

func (p *Provider) Authorize(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	if err := req.Amount.Validate(); err != nil {
		return nil, err
	}

	// AuthorizeOrder authorizes an approved order (hold funds).
	var auth *paypalSDK.AuthorizeOrderResponse
	err := p.withRetry(ctx, func(ctx context.Context) error {
		var e error
		auth, e = p.client.AuthorizeOrder(ctx, req.SourceID, paypalSDK.AuthorizeOrderRequest{})
		if e != nil {
			return errors.Internal("failed to authorize paypal order", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	status := payment.StatusAuthorized
	if strings.EqualFold(auth.Status, "COMPLETED") || strings.EqualFold(auth.Status, "CREATED") {
		status = payment.StatusAuthorized
	}

	return &payment.Transaction{
		ID:             auth.ID,
		Amount:         req.Amount,
		Status:         status,
		SourceID:       req.SourceID,
		Description:    req.Description,
		CreatedAt:      time.Now().UTC(),
		IdempotencyKey: req.IdempotencyKey,
	}, nil
}

func (p *Provider) Capture(ctx context.Context, req *payment.CaptureRequest) (*payment.Transaction, error) {
	var capture *paypalSDK.CaptureOrderResponse
	err := p.withRetry(ctx, func(ctx context.Context) error {
		var e error
		capture, e = p.client.CaptureOrder(ctx, req.TransactionID, paypalSDK.CaptureOrderRequest{})
		if e != nil {
			return errors.Internal("failed to capture paypal authorization", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	amt := req.Amount
	if amt.IsZero() {
		amt = commerce.Zero("USD")
	}

	return &payment.Transaction{
		ID:        capture.ID,
		Amount:    amt,
		Status:    payment.StatusSucceeded,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) Void(ctx context.Context, transactionID string) (*payment.Transaction, error) {
	err := p.withRetry(ctx, func(ctx context.Context) error {
		_, e := p.client.VoidAuthorization(ctx, transactionID)
		if e != nil {
			return errors.Internal("failed to void paypal authorization", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payment.Transaction{
		ID:        transactionID,
		Status:    payment.StatusVoided,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) Refund(ctx context.Context, req *payment.RefundRequest) (*payment.Transaction, error) {
	cur := req.Amount.Currency
	if cur == "" {
		cur = "USD"
	}
	amt := fmt.Sprintf("%d.%02d", req.Amount.Amount/100, abs64(req.Amount.Amount%100))
	if commerce.Decimals(cur) == 0 {
		amt = fmt.Sprintf("%d", req.Amount.Amount)
	}

	var refund *paypalSDK.RefundResponse
	err := p.withRetry(ctx, func(ctx context.Context) error {
		r := paypalSDK.RefundCaptureRequest{
			Amount: &paypalSDK.Money{
				Value:    amt,
				Currency: cur,
			},
			NoteToPayer: req.Reason,
		}
		var e error
		refund, e = p.client.RefundCapture(ctx, req.TransactionID, r)
		if e != nil {
			return errors.Internal("failed to refund paypal transaction", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &payment.Transaction{
		ID:        refund.ID,
		Amount:    req.Amount,
		Status:    payment.StatusRefunded,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (p *Provider) GetTransaction(ctx context.Context, id string) (*payment.Transaction, error) {
	var cap *paypalSDK.CaptureDetailsResponse
	err := p.withRetry(ctx, func(ctx context.Context) error {
		var e error
		cap, e = p.client.CapturedDetail(ctx, id)
		if e != nil {
			return errors.NotFound("transaction not found", e)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	logger.L().DebugContext(ctx, "paypal capture retrieved", "status", cap.Status)

	return &payment.Transaction{
		ID:     cap.ID,
		Status: payment.StatusSucceeded,
	}, nil
}

func (p *Provider) Close() error {
	return nil
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// WebhookVerifier verifies PayPal webhooks via the VerifyWebhookSignature API.
// For unit tests without network, use NewLocalWebhookVerifier.
type WebhookVerifier struct {
	client    *paypalSDK.Client
	webhookID string
	// verifyFn optionally overrides remote verification (tests).
	verifyFn func(ctx context.Context, req *http.Request, webhookID string) error
}

// NewWebhookVerifier creates a PayPal webhook verifier using the live/sandbox client.
func NewWebhookVerifier(cfg payment.Config) (*WebhookVerifier, error) {
	if cfg.PayPalClientID == "" || cfg.PayPalSecret == "" {
		return nil, errors.InvalidArgument("paypal credentials required", nil)
	}
	if cfg.PayPalWebhookID == "" {
		return nil, errors.InvalidArgument("paypal webhook id is required", nil)
	}

	base := paypalSDK.APIBaseLive
	if cfg.PayPalSandbox {
		base = paypalSDK.APIBaseSandBox
	}
	c, err := paypalSDK.NewClient(cfg.PayPalClientID, cfg.PayPalSecret, base)
	if err != nil {
		return nil, errors.Internal("failed to create paypal client", err)
	}
	return &WebhookVerifier{client: c, webhookID: cfg.PayPalWebhookID}, nil
}

// NewLocalWebhookVerifier returns a verifier that checks required PayPal headers
// and an optional shared secret in header X-PayPal-Test-Secret (for fixtures/tests).
func NewLocalWebhookVerifier(expectedSecret string) *WebhookVerifier {
	return &WebhookVerifier{
		webhookID: "local",
		verifyFn: func(ctx context.Context, req *http.Request, webhookID string) error {
			required := []string{
				"PAYPAL-TRANSMISSION-ID",
				"PAYPAL-TRANSMISSION-SIG",
				"PAYPAL-TRANSMISSION-TIME",
				"PAYPAL-AUTH-ALGO",
				"PAYPAL-CERT-URL",
			}
			for _, h := range required {
				if req.Header.Get(h) == "" {
					return payment.ErrInvalidWebhook
				}
			}
			if expectedSecret != "" && req.Header.Get("X-PayPal-Test-Secret") != expectedSecret {
				return payment.ErrInvalidWebhook
			}
			return nil
		},
	}
}

// Verify validates PayPal webhook headers/signature and returns a normalized event.
func (v *WebhookVerifier) Verify(ctx context.Context, payload []byte, headers map[string]string) (*payment.WebhookEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://paypal.local/webhook", strings.NewReader(string(payload)))
	if err != nil {
		return nil, errors.Internal("failed to build webhook request", err)
	}
	for k, val := range headers {
		req.Header.Set(k, val)
	}

	if v.verifyFn != nil {
		if err := v.verifyFn(ctx, req, v.webhookID); err != nil {
			return nil, err
		}
	} else if v.client != nil {
		resp, err := v.client.VerifyWebhookSignature(ctx, req, v.webhookID)
		if err != nil {
			return nil, errors.New("INVALID_WEBHOOK", "invalid webhook signature", err)
		}
		if resp == nil || !strings.EqualFold(resp.VerificationStatus, "SUCCESS") {
			return nil, payment.ErrInvalidWebhook
		}
	} else {
		return nil, errors.Internal("paypal webhook verifier not configured", nil)
	}

	eventType := headers["Event-Type"]
	if eventType == "" {
		eventType = "paypal.webhook"
	}
	eventID := headers["PAYPAL-TRANSMISSION-ID"]
	if eventID == "" {
		eventID = headers["Paypal-Transmission-Id"]
	}

	return &payment.WebhookEvent{
		ID:        eventID,
		Type:      eventType,
		Provider:  "paypal",
		CreatedAt: time.Now().UTC(),
		Raw:       append([]byte(nil), payload...),
		Data: map[string]string{
			"webhook_id": v.webhookID,
		},
	}, nil
}
