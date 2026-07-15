package memory

import (
	"context"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce"
	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/google/uuid"
)

// Ensure compile-time interface compliance.
var _ billing.Service = (*Service)(nil)

// Service implements billing.Service in memory with a built-in plan catalog.
type Service struct {
	subscriptions map[string]*billing.Subscription
	invoices      map[string]*billing.Invoice
	plans         map[string]*billing.Plan
	mu            *concurrency.SmartRWMutex
	// nowFunc allows tests to control "now" for proration.
	nowFunc func() time.Time
}

// DefaultPlans returns the built-in memory plan catalog.
func DefaultPlans() map[string]*billing.Plan {
	return map[string]*billing.Plan{
		"basic_monthly": {
			ID:       "basic_monthly",
			Name:     "Basic Monthly",
			Amount:   commerce.NewMoney(1000, "USD"), // $10.00
			Interval: "month",
		},
		"pro_monthly": {
			ID:       "pro_monthly",
			Name:     "Pro Monthly",
			Amount:   commerce.NewMoney(2900, "USD"), // $29.00
			Interval: "month",
		},
		"pro_yearly": {
			ID:       "pro_yearly",
			Name:     "Pro Yearly",
			Amount:   commerce.NewMoney(29000, "USD"), // $290.00
			Interval: "year",
		},
	}
}

// New creates a new memory billing service with DefaultPlans.
func New() *Service {
	return NewWithPlans(DefaultPlans())
}

// NewWithPlans creates a memory billing service with a custom catalog.
func NewWithPlans(plans map[string]*billing.Plan) *Service {
	cp := make(map[string]*billing.Plan, len(plans))
	for k, v := range plans {
		p := *v
		cp[k] = &p
	}
	return &Service{
		subscriptions: make(map[string]*billing.Subscription),
		invoices:      make(map[string]*billing.Invoice),
		plans:         cp,
		mu:            concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "commerce-billing-memory"}),
		nowFunc:       func() time.Time { return time.Now().UTC() },
	}
}

// SetNowFunc overrides the clock (test helper).
func (s *Service) SetNowFunc(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fn == nil {
		s.nowFunc = func() time.Time { return time.Now().UTC() }
		return
	}
	s.nowFunc = fn
}

func (s *Service) now() time.Time {
	return s.nowFunc()
}

func (s *Service) GetPlan(ctx context.Context, planID string) (*billing.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.plans[planID]
	if !ok {
		return nil, billing.ErrPlanNotFound
	}
	cp := *plan
	return &cp, nil
}

func (s *Service) ListPlans(ctx context.Context) ([]*billing.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*billing.Plan, 0, len(s.plans))
	for _, p := range s.plans {
		cp := *p
		out = append(out, &cp)
	}
	return out, nil
}

func (s *Service) CreateSubscription(ctx context.Context, customerID string, planID string) (*billing.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	plan, ok := s.plans[planID]
	if !ok {
		return nil, billing.ErrPlanNotFound
	}

	id := uuid.New().String()
	now := s.now()
	next := now.AddDate(0, 1, 0)
	if plan.Interval == "year" {
		next = now.AddDate(1, 0, 0)
	}
	sub := &billing.Subscription{
		ID:          id,
		CustomerID:  customerID,
		PlanID:      planID,
		Status:      billing.StatusActive,
		Amount:      plan.Amount,
		Interval:    plan.Interval,
		NextBillAt:  next,
		PeriodStart: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.subscriptions[id] = sub
	cp := *sub
	return &cp, nil
}

func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string) (*billing.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}
	if sub.Status == billing.StatusCanceled {
		return nil, billing.ErrSubscriptionCanceled
	}

	sub.Status = billing.StatusCanceled
	sub.UpdatedAt = s.now()
	cp := *sub
	return &cp, nil
}

func (s *Service) GetSubscription(ctx context.Context, subscriptionID string) (*billing.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}
	cp := *sub
	return &cp, nil
}

func (s *Service) UpgradeSubscription(ctx context.Context, subscriptionID string, newPlanID string) (*billing.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}
	if sub.Status == billing.StatusCanceled {
		return nil, billing.ErrSubscriptionCanceled
	}
	if sub.PlanID == newPlanID {
		return nil, billing.ErrSamePlan
	}

	plan, ok := s.plans[newPlanID]
	if !ok {
		return nil, billing.ErrPlanNotFound
	}

	now := s.now()
	periodStart := sub.PeriodStart
	if periodStart.IsZero() {
		periodStart = sub.CreatedAt
	}
	periodEnd := sub.NextBillAt
	if !periodEnd.After(periodStart) {
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	pr, err := billing.Prorate(sub.Amount, plan.Amount, periodStart, periodEnd, now)
	if err != nil {
		return nil, err
	}

	// Issue a proration invoice when the customer owes a net positive amount.
	if pr.Net.Amount > 0 {
		inv := &billing.Invoice{
			ID:             uuid.New().String(),
			SubscriptionID: sub.ID,
			CustomerID:     sub.CustomerID,
			Amount:         pr.Net,
			Status:         billing.InvoiceOpen,
			IssuedAt:       now,
			Description:    "proration",
		}
		s.invoices[inv.ID] = inv
	}

	sub.PlanID = newPlanID
	sub.Amount = plan.Amount
	sub.Interval = plan.Interval
	sub.Status = billing.StatusActive
	sub.UpdatedAt = now
	cp := *sub
	return &cp, nil
}

func (s *Service) MarkPastDue(ctx context.Context, subscriptionID string) (*billing.Subscription, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}
	if sub.Status == billing.StatusCanceled {
		return nil, billing.ErrSubscriptionCanceled
	}

	sub.Status = billing.StatusPastDue
	sub.UpdatedAt = s.now()
	cp := *sub
	return &cp, nil
}

// ProcessDunning transitions open invoices for the subscription to past_due
// and marks the subscription StatusPastDue.
func (s *Service) ProcessDunning(ctx context.Context, subscriptionID string) (*billing.DunningResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}
	if sub.Status == billing.StatusCanceled {
		return nil, billing.ErrSubscriptionCanceled
	}

	now := s.now()
	var transitioned []*billing.Invoice
	for _, inv := range s.invoices {
		if inv.SubscriptionID != subscriptionID {
			continue
		}
		if inv.Status != billing.InvoiceOpen {
			continue
		}
		inv.Status = billing.InvoicePastDue
		cp := *inv
		transitioned = append(transitioned, &cp)
	}

	sub.Status = billing.StatusPastDue
	sub.UpdatedAt = now
	subCP := *sub
	return &billing.DunningResult{
		Subscription: &subCP,
		Invoices:     transitioned,
	}, nil
}

func (s *Service) CreateInvoice(ctx context.Context, customerID string, amount commerce.Money) (*billing.Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := amount.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	inv := &billing.Invoice{
		ID:         id,
		CustomerID: customerID,
		Amount:     amount,
		Status:     billing.InvoiceOpen,
		IssuedAt:   s.now(),
	}
	s.invoices[id] = inv
	cp := *inv
	return &cp, nil
}

// CreateSubscriptionInvoice creates an open invoice linked to a subscription (test/helper).
func (s *Service) CreateSubscriptionInvoice(ctx context.Context, subscriptionID string, amount commerce.Money) (*billing.Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := amount.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, billing.ErrSubscriptionNotFound
	}

	id := uuid.New().String()
	inv := &billing.Invoice{
		ID:             id,
		SubscriptionID: subscriptionID,
		CustomerID:     sub.CustomerID,
		Amount:         amount,
		Status:         billing.InvoiceOpen,
		IssuedAt:       s.now(),
	}
	s.invoices[id] = inv
	cp := *inv
	return &cp, nil
}

func (s *Service) ListInvoices(ctx context.Context, customerID string) ([]*billing.Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*billing.Invoice, 0)
	for _, inv := range s.invoices {
		if customerID != "" && inv.CustomerID != customerID {
			continue
		}
		cp := *inv
		out = append(out, &cp)
	}
	return out, nil
}
