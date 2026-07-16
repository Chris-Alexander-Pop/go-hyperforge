package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Item is a line item in a cart.
type Item struct {
	SKU         string
	Qty         int64
	AmountMinor int64
	Currency    string
}

// Cart is a shopping cart.
type Cart struct {
	ID        string
	UserID    string
	Items     []Item
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CheckoutResult is a snapshot returned at checkout.
type CheckoutResult struct {
	CartID     string
	UserID     string
	Items      []Item
	Currency   string
	TotalMinor int64
	CheckedOut time.Time
}

// AddItemInput adds or increases a cart line.
type AddItemInput struct {
	SKU         string
	Qty         int64
	AmountMinor int64
	Currency    string
}

// Store is an in-memory cart store.
type Store struct {
	mu    sync.RWMutex
	carts map[string]*Cart
}

// New creates an empty in-memory cart store.
func New() *Store {
	return &Store{carts: make(map[string]*Cart)}
}

// Create creates a new empty cart.
func (s *Store) Create(ctx context.Context, userID string) (*Cart, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	c := &Cart{
		ID:        uuid.NewString(),
		UserID:    userID,
		Items:     []Item{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.carts[c.ID] = c
	return cloneCart(c), nil
}

// Get returns a cart by ID.
func (s *Store) Get(ctx context.Context, id string) (*Cart, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.carts[id]
	if !ok {
		return nil, errors.NotFound("cart not found", nil)
	}
	return cloneCart(c), nil
}

// AddItem adds qty to an existing SKU or inserts a new line.
func (s *Store) AddItem(ctx context.Context, cartID string, in AddItemInput) (*Cart, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cartID == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	if in.SKU == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if in.Qty <= 0 {
		return nil, errors.InvalidArgument("qty must be positive", nil)
	}
	if in.Currency == "" {
		return nil, errors.InvalidArgument("currency is required", nil)
	}
	money := commerce.NewMoney(in.AmountMinor, in.Currency)

	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.carts[cartID]
	if !ok {
		return nil, errors.NotFound("cart not found", nil)
	}
	for i := range c.Items {
		if c.Items[i].SKU == in.SKU {
			if !commerce.NewMoney(c.Items[i].AmountMinor, c.Items[i].Currency).SameCurrency(money) {
				return nil, errors.InvalidArgument("currency mismatch for existing sku", nil)
			}
			c.Items[i].Qty += in.Qty
			c.Items[i].AmountMinor = money.Amount
			c.UpdatedAt = time.Now().UTC()
			return cloneCart(c), nil
		}
	}
	c.Items = append(c.Items, Item{
		SKU:         in.SKU,
		Qty:         in.Qty,
		AmountMinor: money.Amount,
		Currency:    money.Currency,
	})
	c.UpdatedAt = time.Now().UTC()
	return cloneCart(c), nil
}

// RemoveItem removes a SKU from the cart.
func (s *Store) RemoveItem(ctx context.Context, cartID, sku string) (*Cart, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cartID == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.carts[cartID]
	if !ok {
		return nil, errors.NotFound("cart not found", nil)
	}
	found := false
	next := make([]Item, 0, len(c.Items))
	for _, it := range c.Items {
		if it.SKU == sku {
			found = true
			continue
		}
		next = append(next, it)
	}
	if !found {
		return nil, errors.NotFound("cart item not found", nil)
	}
	c.Items = next
	c.UpdatedAt = time.Now().UTC()
	return cloneCart(c), nil
}

// Checkout snapshots the cart into an order-like payload.
func (s *Store) Checkout(ctx context.Context, cartID string) (*CheckoutResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if cartID == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.carts[cartID]
	if !ok {
		return nil, errors.NotFound("cart not found", nil)
	}
	if len(c.Items) == 0 {
		return nil, errors.FailedPrecondition("cart is empty", nil)
	}

	currency := c.Items[0].Currency
	total := commerce.Zero(currency)
	items := make([]Item, len(c.Items))
	copy(items, c.Items)
	for _, it := range items {
		line := commerce.NewMoney(it.AmountMinor*it.Qty, it.Currency)
		if !line.SameCurrency(total) {
			return nil, errors.InvalidArgument("cart items have mixed currencies", nil)
		}
		var err error
		total, err = total.Add(line)
		if err != nil {
			return nil, err
		}
	}

	return &CheckoutResult{
		CartID:     c.ID,
		UserID:     c.UserID,
		Items:      items,
		Currency:   currency,
		TotalMinor: total.Amount,
		CheckedOut: time.Now().UTC(),
	}, nil
}

func cloneCart(c *Cart) *Cart {
	cp := *c
	if c.Items != nil {
		cp.Items = make([]Item, len(c.Items))
		copy(cp.Items, c.Items)
	}
	return &cp
}
