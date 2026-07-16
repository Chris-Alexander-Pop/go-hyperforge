package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// Product is a catalog product.
type Product struct {
	ID          string
	Name        string
	SKU         string
	PriceMinor  int64
	Currency    string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateInput creates a product.
type CreateInput struct {
	Name        string
	SKU         string
	PriceMinor  int64
	Currency    string
	Description string
}

// UpdateInput updates mutable product fields.
type UpdateInput struct {
	Name        string
	SKU         string
	PriceMinor  int64
	Currency    string
	Description string
}

// Store is an in-memory product catalog.
type Store struct {
	mu       sync.RWMutex
	products map[string]*Product
	bySKU    map[string]string
}

// New creates an empty product store.
func New() *Store {
	return &Store{
		products: make(map[string]*Product),
		bySKU:    make(map[string]string),
	}
}

// Create inserts a new product.
func (s *Store) Create(ctx context.Context, in CreateInput) (*Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, errors.InvalidArgument("name is required", nil)
	}
	if in.SKU == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if in.Currency == "" {
		return nil, errors.InvalidArgument("currency is required", nil)
	}
	money := commerce.NewMoney(in.PriceMinor, in.Currency)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.bySKU[in.SKU]; exists {
		return nil, errors.Conflict("sku already exists", nil)
	}
	now := time.Now().UTC()
	p := &Product{
		ID:          uuid.NewString(),
		Name:        in.Name,
		SKU:         in.SKU,
		PriceMinor:  money.Amount,
		Currency:    money.Currency,
		Description: in.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.products[p.ID] = p
	s.bySKU[p.SKU] = p.ID
	return cloneProduct(p), nil
}

// Get returns a product by ID.
func (s *Store) Get(ctx context.Context, id string) (*Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	if !ok {
		return nil, errors.NotFound("product not found", nil)
	}
	return cloneProduct(p), nil
}

// List returns all products.
func (s *Store) List(ctx context.Context) ([]*Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Product, 0, len(s.products))
	for _, p := range s.products {
		out = append(out, cloneProduct(p))
	}
	return out, nil
}

// Update replaces product fields.
func (s *Store) Update(ctx context.Context, id string, in UpdateInput) (*Product, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	if in.Name == "" {
		return nil, errors.InvalidArgument("name is required", nil)
	}
	if in.SKU == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if in.Currency == "" {
		return nil, errors.InvalidArgument("currency is required", nil)
	}
	money := commerce.NewMoney(in.PriceMinor, in.Currency)

	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return nil, errors.NotFound("product not found", nil)
	}
	if otherID, exists := s.bySKU[in.SKU]; exists && otherID != id {
		return nil, errors.Conflict("sku already exists", nil)
	}
	delete(s.bySKU, p.SKU)
	p.Name = in.Name
	p.SKU = in.SKU
	p.PriceMinor = money.Amount
	p.Currency = money.Currency
	p.Description = in.Description
	p.UpdatedAt = time.Now().UTC()
	s.bySKU[p.SKU] = p.ID
	return cloneProduct(p), nil
}

// Delete removes a product.
func (s *Store) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return errors.InvalidArgument("id is required", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.products[id]
	if !ok {
		return errors.NotFound("product not found", nil)
	}
	delete(s.bySKU, p.SKU)
	delete(s.products, id)
	return nil
}

func cloneProduct(p *Product) *Product {
	cp := *p
	return &cp
}
