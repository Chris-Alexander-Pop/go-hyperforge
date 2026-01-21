package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/billing"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Service implements billing.Service in memory.
type Service struct {
	subscriptions map[string]*billing.Subscription
	invoices      map[string]*billing.Invoice
	mu            sync.RWMutex
}

// New creates a new memory billing service.
func New() *Service {
	return &Service{
		subscriptions: make(map[string]*billing.Subscription),
		invoices:      make(map[string]*billing.Invoice),
	}
}

func (s *Service) CreateSubscription(ctx context.Context, customerID string, planID string) (*billing.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	sub := &billing.Subscription{
		ID:         id,
		CustomerID: customerID,
		PlanID:     planID,
		Status:     billing.StatusActive,
		Amount:     10.0, // Mock amount per plan
		Currency:   "USD",
		Interval:   "month",
		NextBillAt: time.Now().AddDate(0, 1, 0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.subscriptions[id] = sub
	return sub, nil
}

func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string) (*billing.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, errors.NotFound("subscription not found", nil)
	}

	sub.Status = billing.StatusCanceled
	sub.UpdatedAt = time.Now()

	return sub, nil
}

func (s *Service) GetSubscription(ctx context.Context, subscriptionID string) (*billing.Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, ok := s.subscriptions[subscriptionID]
	if !ok {
		return nil, errors.NotFound("subscription not found", nil)
	}
	return sub, nil
}

func (s *Service) CreateInvoice(ctx context.Context, customerID string, amount float64, currency string) (*billing.Invoice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	inv := &billing.Invoice{
		ID:         id,
		CustomerID: customerID,
		Amount:     amount,
		Currency:   strings.ToUpper(currency),
		Status:     "open",
		IssuedAt:   time.Now(),
	}

	s.invoices[id] = inv
	return inv, nil
}
