package tests

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment/adapters/memory"
	paypaladapter "github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment/adapters/paypal"
	stripeadapter "github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment/adapters/stripe"
	"github.com/chris-alexander-pop/system-design-library/pkg/events"
	eventsmemory "github.com/chris-alexander-pop/system-design-library/pkg/events/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
	"github.com/stripe/stripe-go/v76"
)

type PaymentTestSuite struct {
	test.Suite
	provider payment.Authorizer
}

func (s *PaymentTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.provider = memory.New()
}

func (s *PaymentTestSuite) TestCharge() {
	req := &payment.ChargeRequest{
		Amount:   commerce.NewMoney(10000, "USD"),
		SourceID: "tok_visa",
	}

	tx, err := s.provider.Charge(s.Ctx, req)
	s.NoError(err)
	s.NotNil(tx)
	s.Equal(payment.StatusSucceeded, tx.Status)
	s.Equal(int64(10000), tx.Amount.Amount)
	s.NotEmpty(tx.ID)
}

func (s *PaymentTestSuite) TestChargeFail() {
	req := &payment.ChargeRequest{
		Amount:   commerce.NewMoney(10000, "USD"),
		SourceID: "fail_card",
	}

	_, err := s.provider.Charge(s.Ctx, req)
	s.Error(err)
	s.Equal(payment.ErrDeclined, err)
}

func (s *PaymentTestSuite) TestIdempotency() {
	req := &payment.ChargeRequest{
		Amount:         commerce.NewMoney(5000, "USD"),
		SourceID:       "tok_visa",
		IdempotencyKey: "idem-1",
	}
	tx1, err := s.provider.Charge(s.Ctx, req)
	s.NoError(err)
	tx2, err := s.provider.Charge(s.Ctx, req)
	s.NoError(err)
	s.Equal(tx1.ID, tx2.ID)

	conflict := &payment.ChargeRequest{
		Amount:         commerce.NewMoney(6000, "USD"),
		SourceID:       "tok_visa",
		IdempotencyKey: "idem-1",
	}
	_, err = s.provider.Charge(s.Ctx, conflict)
	s.Equal(payment.ErrIdempotencyConflict, err)
}

func (s *PaymentTestSuite) TestAuthorizeCaptureVoid() {
	auth, err := s.provider.Authorize(s.Ctx, &payment.ChargeRequest{
		Amount:   commerce.NewMoney(2000, "USD"),
		SourceID: "tok_visa",
	})
	s.NoError(err)
	s.Equal(payment.StatusAuthorized, auth.Status)

	captured, err := s.provider.Capture(s.Ctx, &payment.CaptureRequest{TransactionID: auth.ID})
	s.NoError(err)
	s.Equal(payment.StatusSucceeded, captured.Status)

	auth2, err := s.provider.Authorize(s.Ctx, &payment.ChargeRequest{
		Amount:   commerce.NewMoney(1500, "USD"),
		SourceID: "tok_visa",
	})
	s.NoError(err)
	voided, err := s.provider.Void(s.Ctx, auth2.ID)
	s.NoError(err)
	s.Equal(payment.StatusVoided, voided.Status)
}

func (s *PaymentTestSuite) TestRefund() {
	chargeTx, err := s.provider.Charge(s.Ctx, &payment.ChargeRequest{
		Amount:   commerce.NewMoney(5000, "USD"),
		SourceID: "tok_visa",
	})
	s.NoError(err)

	refundTx, err := s.provider.Refund(s.Ctx, &payment.RefundRequest{
		TransactionID: chargeTx.ID,
		Amount:        commerce.NewMoney(5000, "USD"),
	})
	s.NoError(err)
	s.NotNil(refundTx)
	s.Equal(payment.StatusRefunded, refundTx.Status)
}

func (s *PaymentTestSuite) TestEventedChargePublishes() {
	bus := eventsmemory.New(events.Config{})
	defer bus.Close()

	var got []events.Event
	var mu sync.Mutex
	_, err := bus.Subscribe(s.Ctx, payment.TopicPayment, func(ctx context.Context, e events.Event) error {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
		return nil
	})
	s.NoError(err)

	evented := payment.NewEventedProvider(memory.New(), bus)
	_, err = evented.Charge(s.Ctx, &payment.ChargeRequest{
		Amount:   commerce.NewMoney(1000, "USD"),
		SourceID: "tok_visa",
	})
	s.NoError(err)

	mu.Lock()
	defer mu.Unlock()
	s.Len(got, 1)
	s.Equal(payment.EventTypeChargeSucceeded, got[0].Type)
}

func (s *PaymentTestSuite) TestEventedNilBus() {
	evented := payment.NewEventedProvider(memory.New(), nil)
	_, err := evented.Charge(s.Ctx, &payment.ChargeRequest{
		Amount:   commerce.NewMoney(1000, "USD"),
		SourceID: "tok_visa",
	})
	s.NoError(err)
}

func TestPaymentSuite(t *testing.T) {
	test.Run(t, new(PaymentTestSuite))
}

func TestStripeWebhookVerify(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(fmt.Sprintf(`{
  "id": "evt_test_webhook",
  "object": "event",
  "api_version": "%s",
  "type": "payment_intent.succeeded",
  "created": %d,
  "data": {"object": {"id": "pi_test"}}
}`, stripe.APIVersion, time.Now().Unix()))

	ts := time.Now()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", ts.Unix())))
	mac.Write([]byte("."))
	mac.Write(payload)
	sig := hex.EncodeToString(mac.Sum(nil))
	header := fmt.Sprintf("t=%d,v1=%s", ts.Unix(), sig)

	v, err := stripeadapter.NewWebhookVerifier(secret)
	if err != nil {
		t.Fatal(err)
	}
	evt, err := v.Verify(context.Background(), payload, map[string]string{
		"Stripe-Signature": header,
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if evt.ID != "evt_test_webhook" {
		t.Fatalf("id=%s", evt.ID)
	}
	if evt.Provider != "stripe" {
		t.Fatalf("provider=%s", evt.Provider)
	}

	_, err = v.Verify(context.Background(), payload, map[string]string{
		"Stripe-Signature": "t=1,v1=deadbeef",
	})
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestPayPalLocalWebhookVerify(t *testing.T) {
	v := paypaladapter.NewLocalWebhookVerifier("test-secret")
	payload := []byte(`{"id":"WH-1","event_type":"PAYMENT.CAPTURE.COMPLETED"}`)

	_, err := v.Verify(context.Background(), payload, map[string]string{
		"PAYPAL-TRANSMISSION-ID":   "tx-1",
		"PAYPAL-TRANSMISSION-SIG":  "sig",
		"PAYPAL-TRANSMISSION-TIME": time.Now().UTC().Format(time.RFC3339),
		"PAYPAL-AUTH-ALGO":         "SHA256withRSA",
		"PAYPAL-CERT-URL":          "https://example.com/cert",
		"X-PayPal-Test-Secret":     "test-secret",
		"Event-Type":               "PAYMENT.CAPTURE.COMPLETED",
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	_, err = v.Verify(context.Background(), payload, map[string]string{
		"PAYPAL-TRANSMISSION-ID": "tx-1",
		"X-PayPal-Test-Secret":   "wrong",
	})
	if err == nil {
		t.Fatal("expected missing headers / bad secret to fail")
	}
}
