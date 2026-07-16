package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Status is the lifecycle state of an order.
type Status string

const (
	StatusPending   Status = "pending"
	StatusPaid      Status = "paid"
	StatusCancelled Status = "cancelled"
	StatusFulfilled Status = "fulfilled"
)

// Item is a line item on an order.
type Item struct {
	SKU         string
	Qty         int64
	AmountMinor int64
	Currency    string
}

// Order is a commerce order record.
type Order struct {
	ID         string
	CustomerID string
	Items      []Item
	Currency   string
	TotalMinor int64
	Status     Status
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateInput is the payload to create an order.
type CreateInput struct {
	CustomerID string
	Items      []Item
	Currency   string
}

// Store is an in-memory order store.
type Store struct {
	mu     sync.RWMutex
	orders map[string]*Order
}

// New creates an empty in-memory order store.
func New() *Store {
	return &Store{orders: make(map[string]*Order)}
}

// Create validates and stores a new pending order.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.CustomerID == "" {
		return nil, errors.InvalidArgument("customer_id is required", nil)
	}
	if in.Currency == "" {
		return nil, errors.InvalidArgument("currency is required", nil)
	}
	if len(in.Items) == 0 {
		return nil, errors.InvalidArgument("items are required", nil)
	}

	currency := commerce.NewMoney(0, in.Currency).Currency
	total := commerce.Zero(currency)
	items := make([]Item, 0, len(in.Items))
	for _, it := range in.Items {
		if it.SKU == "" {
			return nil, errors.InvalidArgument("items[].sku is required", nil)
		}
		if it.Qty <= 0 {
			return nil, errors.InvalidArgument("items[].qty must be positive", nil)
		}
		itemCurrency := it.Currency
		if itemCurrency == "" {
			itemCurrency = currency
		}
		line := commerce.NewMoney(it.AmountMinor, itemCurrency)
		if !line.SameCurrency(total) {
			return nil, errors.InvalidArgument("item currency must match order currency", nil)
		}
		// amount_minor is per-unit; total = sum(amount * qty)
		lineTotal := commerce.NewMoney(line.Amount*it.Qty, line.Currency)
		var err error
		total, err = total.Add(lineTotal)
		if err != nil {
			return nil, err
		}
		items = append(items, Item{
			SKU:         it.SKU,
			Qty:         it.Qty,
			AmountMinor: line.Amount,
			Currency:    line.Currency,
		})
	}

	now := time.Now().UTC()
	o := &Order{
		ID:         uuid.NewString(),
		CustomerID: in.CustomerID,
		Items:      items,
		Currency:   currency,
		TotalMinor: total.Amount,
		Status:     StatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[o.ID] = o
	return cloneOrder(o), nil
}

// Get returns an order by ID.
func (s *Store) Get(ctx context.Context, id string) (*Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, errors.NotFound("order not found", nil)
	}
	return cloneOrder(o), nil
}

// List returns all orders.
func (s *Store) List(ctx context.Context) ([]*Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Order, 0, len(s.orders))
	for _, o := range s.orders {
		out = append(out, cloneOrder(o))
	}
	return out, nil
}

// Cancel marks a pending/paid order as cancelled.
func (s *Store) Cancel(ctx context.Context, id string) (*Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, errors.NotFound("order not found", nil)
	}
	switch o.Status {
	case StatusCancelled:
		return nil, errors.Conflict("order already cancelled", nil)
	case StatusFulfilled:
		return nil, errors.FailedPrecondition("fulfilled order cannot be cancelled", nil)
	}
	o.Status = StatusCancelled
	o.UpdatedAt = time.Now().UTC()
	return cloneOrder(o), nil
}

func cloneOrder(o *Order) *Order {
	cp := *o
	if o.Items != nil {
		cp.Items = make([]Item, len(o.Items))
		copy(cp.Items, o.Items)
	}
	return &cp
}
