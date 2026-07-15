package tests

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/test"
)

type BillingTestSuite struct {
	test.Suite
	service *memory.Service
}

func (s *BillingTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.service = memory.New()
}

func (s *BillingTestSuite) TestPlanCatalog() {
	plans, err := s.service.ListPlans(s.Ctx)
	s.NoError(err)
	s.GreaterOrEqual(len(plans), 3)

	plan, err := s.service.GetPlan(s.Ctx, "basic_monthly")
	s.NoError(err)
	s.Equal(int64(1000), plan.Amount.Amount)
	s.Equal("USD", plan.Amount.Currency)

	_, err = s.service.GetPlan(s.Ctx, "missing")
	s.Equal(billing.ErrPlanNotFound, err)
}

func (s *BillingTestSuite) TestSubscriptionLifecycle() {
	custID := "cust_123"
	planID := "basic_monthly"

	sub, err := s.service.CreateSubscription(s.Ctx, custID, planID)
	s.NoError(err)
	s.NotNil(sub)
	s.Equal(billing.StatusActive, sub.Status)
	s.Equal(custID, sub.CustomerID)
	s.Equal(int64(1000), sub.Amount.Amount)

	got, err := s.service.GetSubscription(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(sub.ID, got.ID)

	canceled, err := s.service.CancelSubscription(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(billing.StatusCanceled, canceled.Status)
}

func (s *BillingTestSuite) TestUpgradeProrationAndPastDue() {
	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s.service.SetNowFunc(func() time.Time { return fixed })

	sub, err := s.service.CreateSubscription(s.Ctx, "cust_up", "basic_monthly")
	s.NoError(err)

	// Mid-cycle upgrade at day 15 of a 31-day January period.
	mid := fixed.AddDate(0, 0, 15)
	s.service.SetNowFunc(func() time.Time { return mid })

	upgraded, err := s.service.UpgradeSubscription(s.Ctx, sub.ID, "pro_monthly")
	s.NoError(err)
	s.Equal("pro_monthly", upgraded.PlanID)
	s.Equal(int64(2900), upgraded.Amount.Amount)
	s.Equal(billing.StatusActive, upgraded.Status)

	invoices, err := s.service.ListInvoices(s.Ctx, "cust_up")
	s.NoError(err)
	s.Len(invoices, 1)
	s.Equal("proration", invoices[0].Description)
	s.Equal(billing.InvoiceOpen, invoices[0].Status)
	s.Equal(sub.ID, invoices[0].SubscriptionID)
	s.True(invoices[0].Amount.Amount > 0)

	pastDue, err := s.service.MarkPastDue(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(billing.StatusPastDue, pastDue.Status)

	_, err = s.service.UpgradeSubscription(s.Ctx, sub.ID, "pro_monthly")
	s.Equal(billing.ErrSamePlan, err)
}

func (s *BillingTestSuite) TestProcessDunning() {
	sub, err := s.service.CreateSubscription(s.Ctx, "cust_dun", "basic_monthly")
	s.NoError(err)

	inv, err := s.service.CreateSubscriptionInvoice(s.Ctx, sub.ID, commerce.NewMoney(1000, "USD"))
	s.NoError(err)
	s.Equal(billing.InvoiceOpen, inv.Status)

	// Unrelated open invoice should not be touched.
	_, err = s.service.CreateInvoice(s.Ctx, "cust_dun", commerce.NewMoney(500, "USD"))
	s.NoError(err)

	result, err := s.service.ProcessDunning(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(billing.StatusPastDue, result.Subscription.Status)
	s.Len(result.Invoices, 1)
	s.Equal(inv.ID, result.Invoices[0].ID)
	s.Equal(billing.InvoicePastDue, result.Invoices[0].Status)

	all, err := s.service.ListInvoices(s.Ctx, "cust_dun")
	s.NoError(err)
	var openCount, pastDueCount int
	for _, i := range all {
		switch i.Status {
		case billing.InvoiceOpen:
			openCount++
		case billing.InvoicePastDue:
			pastDueCount++
		}
	}
	s.Equal(1, openCount)
	s.Equal(1, pastDueCount)
}

func (s *BillingTestSuite) TestProrateHelper() {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC) // ~halfway

	oldAmt := commerce.NewMoney(1000, "USD")
	newAmt := commerce.NewMoney(3000, "USD")
	pr, err := billing.Prorate(oldAmt, newAmt, start, end, mid)
	s.NoError(err)
	s.InDelta(0.5, pr.Fraction, 0.02)
	s.True(pr.Credit.Amount > 400 && pr.Credit.Amount < 600)
	s.True(pr.Charge.Amount > 1400 && pr.Charge.Amount < 1600)
	s.Equal(pr.Charge.Amount-pr.Credit.Amount, pr.Net.Amount)

	_, err = billing.Prorate(oldAmt, commerce.NewMoney(100, "EUR"), start, end, mid)
	s.Equal(billing.ErrCurrencyMismatch, err)

	_, err = billing.Prorate(oldAmt, newAmt, end, start, mid)
	s.Equal(billing.ErrInvalidPeriod, err)
}

func (s *BillingTestSuite) TestCreateInvoice() {
	inv, err := s.service.CreateInvoice(s.Ctx, "cust_123", commerce.NewMoney(9900, "USD"))
	s.NoError(err)
	s.NotNil(inv)
	s.Equal(int64(9900), inv.Amount.Amount)
	s.Equal("USD", inv.Amount.Currency)
	s.Equal(billing.InvoiceOpen, inv.Status)
}

func (s *BillingTestSuite) TestUnknownPlanOnCreate() {
	_, err := s.service.CreateSubscription(s.Ctx, "cust", "nope")
	s.Equal(billing.ErrPlanNotFound, err)
}

func TestBillingSuite(t *testing.T) {
	test.Run(t, new(BillingTestSuite))
}
