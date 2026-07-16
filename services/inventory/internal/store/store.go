package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// SKU holds inventory quantity and reserved units.
type SKU struct {
	SKU       string
	Quantity  int64
	Reserved  int64
	UpdatedAt time.Time
}

// Available returns unreserved stock.
func (s SKU) Available() int64 {
	return s.Quantity - s.Reserved
}

// Store is an in-memory inventory store.
type Store struct {
	mu   sync.RWMutex
	skus map[string]*SKU
}

// New creates an empty inventory store.
func New() *Store {
	return &Store{skus: make(map[string]*SKU)}
}

// Upsert sets absolute quantity for a SKU (creates if missing).
func (s *Store) Upsert(ctx context.Context, sku string, quantity int64) (*SKU, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if quantity < 0 {
		return nil, errors.InvalidArgument("quantity must be non-negative", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.skus[sku]
	if !ok {
		rec = &SKU{SKU: sku}
		s.skus[sku] = rec
	}
	if quantity < rec.Reserved {
		return nil, errors.FailedPrecondition("quantity cannot be less than reserved", nil)
	}
	rec.Quantity = quantity
	rec.UpdatedAt = time.Now().UTC()
	return cloneSKU(rec), nil
}

// Get returns a SKU record.
func (s *Store) Get(ctx context.Context, sku string) (*SKU, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.skus[sku]
	if !ok {
		return nil, errors.NotFound("sku not found", nil)
	}
	return cloneSKU(rec), nil
}

// Reserve holds qty units against available stock.
func (s *Store) Reserve(ctx context.Context, sku string, qty int64) (*SKU, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if qty <= 0 {
		return nil, errors.InvalidArgument("qty must be positive", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.skus[sku]
	if !ok {
		return nil, errors.NotFound("sku not found", nil)
	}
	if rec.Available() < qty {
		return nil, errors.FailedPrecondition("insufficient available inventory", nil)
	}
	rec.Reserved += qty
	rec.UpdatedAt = time.Now().UTC()
	return cloneSKU(rec), nil
}

// Release frees previously reserved qty.
func (s *Store) Release(ctx context.Context, sku string, qty int64) (*SKU, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if qty <= 0 {
		return nil, errors.InvalidArgument("qty must be positive", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.skus[sku]
	if !ok {
		return nil, errors.NotFound("sku not found", nil)
	}
	if rec.Reserved < qty {
		return nil, errors.FailedPrecondition("cannot release more than reserved", nil)
	}
	rec.Reserved -= qty
	rec.UpdatedAt = time.Now().UTC()
	return cloneSKU(rec), nil
}

func cloneSKU(s *SKU) *SKU {
	cp := *s
	return &cp
}
