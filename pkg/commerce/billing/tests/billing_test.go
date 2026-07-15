package tests

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
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

func (s *BillingTestSuite) TestUpgradeAndPastDue() {
	sub, err := s.service.CreateSubscription(s.Ctx, "cust_up", "basic_monthly")
	s.NoError(err)

	upgraded, err := s.service.UpgradeSubscription(s.Ctx, sub.ID, "pro_monthly")
	s.NoError(err)
	s.Equal("pro_monthly", upgraded.PlanID)
	s.Equal(int64(2900), upgraded.Amount.Amount)
	s.Equal(billing.StatusActive, upgraded.Status)

	pastDue, err := s.service.MarkPastDue(s.Ctx, sub.ID)
	s.NoError(err)
	s.Equal(billing.StatusPastDue, pastDue.Status)

	_, err = s.service.UpgradeSubscription(s.Ctx, sub.ID, "pro_monthly")
	s.Equal(billing.ErrSamePlan, err)
}

func (s *BillingTestSuite) TestCreateInvoice() {
	inv, err := s.service.CreateInvoice(s.Ctx, "cust_123", commerce.NewMoney(9900, "USD"))
	s.NoError(err)
	s.NotNil(inv)
	s.Equal(int64(9900), inv.Amount.Amount)
	s.Equal("USD", inv.Amount.Currency)
	s.Equal("open", inv.Status)
}

func (s *BillingTestSuite) TestUnknownPlanOnCreate() {
	_, err := s.service.CreateSubscription(s.Ctx, "cust", "nope")
	s.Equal(billing.ErrPlanNotFound, err)
}

func TestBillingSuite(t *testing.T) {
	test.Run(t, new(BillingTestSuite))
}
