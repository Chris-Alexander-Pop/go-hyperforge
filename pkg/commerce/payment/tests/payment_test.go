package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type PaymentTestSuite struct {
	test.Suite
	provider payment.Provider
}

func (s *PaymentTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.provider = memory.New()
}

func (s *PaymentTestSuite) TestCharge() {
	req := &payment.ChargeRequest{
		Amount:   100.0,
		Currency: "USD",
		SourceID: "tok_visa",
	}

	tx, err := s.provider.Charge(s.Ctx, req)
	s.NoError(err)
	s.NotNil(tx)
	s.Equal(payment.StatusSucceeded, tx.Status)
	s.NotEmpty(tx.ID)
}

func (s *PaymentTestSuite) TestChargeFail() {
	req := &payment.ChargeRequest{
		Amount:   100.0,
		Currency: "USD",
		SourceID: "fail_card",
	}

	_, err := s.provider.Charge(s.Ctx, req)
	s.Error(err)
	s.Equal(payment.ErrDeclined, err)
}

func (s *PaymentTestSuite) TestRefund() {
	// First charge
	req := &payment.ChargeRequest{
		Amount:   50.0,
		Currency: "USD",
		SourceID: "tok_visa",
	}
	chargeTx, err := s.provider.Charge(s.Ctx, req)
	s.NoError(err)

	// Then refund
	refundReq := &payment.RefundRequest{
		TransactionID: chargeTx.ID,
		Amount:        50.0,
	}
	refundTx, err := s.provider.Refund(s.Ctx, refundReq)
	s.NoError(err)
	s.NotNil(refundTx)
	s.Equal(payment.StatusRefunded, refundTx.Status)
}

func TestPaymentSuite(t *testing.T) {
	test.Run(t, new(PaymentTestSuite))
}
