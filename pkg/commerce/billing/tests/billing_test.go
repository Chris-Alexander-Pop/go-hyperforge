package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type BillingTestSuite struct {
	test.Suite
	service billing.Service
}

func (s *BillingTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.service = memory.New()
}

func (s *BillingTestSuite) TestSubscriptionLifecycle() {
	custID := "cust_123"
	planID := "basic_monthly"

	// Create
	sub, err := s.service.CreateSubscription(s.Ctx, custID, planID)
	s.NoError(err)
	s.NotNil(sub)
	s.Equal(billing.StatusActive, sub.Status)
	s.Equal(custID, sub.CustomerID)

	// Get
	got, err := s.service.GetSubscription(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(sub.ID, got.ID)

	// Cancel
	canceled, err := s.service.CancelSubscription(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(billing.StatusCanceled, canceled.Status)
}

func (s *BillingTestSuite) TestCreateInvoice() {
	inv, err := s.service.CreateInvoice(s.Ctx, "cust_123", 99.0, "USD")
	s.NoError(err)
	s.NotNil(inv)
	s.Equal(99.0, inv.Amount)
	s.Equal("open", inv.Status)
}

func TestBillingSuite(t *testing.T) {
	test.Run(t, new(BillingTestSuite))
}
