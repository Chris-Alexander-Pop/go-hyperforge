package store

import (
	"context"
	"sync"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/google/uuid"
)

// PriceRule is a unit price for a SKU.
type PriceRule struct {
	ID          string
	SKU         string
	Name        string
	AmountMinor int64
	Currency    string
	CreatedAt   time.Time
}

// Quote is a computed price for a quantity.
type Quote struct {
	SKU         string
	Qty         int64
	UnitMinor   int64
	TotalMinor  int64
	Currency    string
	PriceRuleID string
}

// CreateInput creates a price rule.
type CreateInput struct {
	SKU         string
	Name        string
	AmountMinor int64
	Currency    string
}

// Store is an in-memory price rule store.
type Store struct {
	mu    sync.RWMutex
	rules map[string]*PriceRule
	bySKU map[string]string // sku -> latest rule id
}

// New creates an empty price store.
func New() *Store {
	return &Store{
		rules: make(map[string]*PriceRule),
		bySKU: make(map[string]string),
	}
}

// Create inserts a price rule (latest per SKU wins for quotes).
func (s *Store) Create(ctx context.Context, in CreateInput) (*PriceRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if in.SKU == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if in.Currency == "" {
		return nil, errors.InvalidArgument("currency is required", nil)
	}
	money := commerce.NewMoney(in.AmountMinor, in.Currency)
	rule := &PriceRule{
		ID:          uuid.NewString(),
		SKU:         in.SKU,
		Name:        in.Name,
		AmountMinor: money.Amount,
		Currency:    money.Currency,
		CreatedAt:   time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.ID] = rule
	s.bySKU[rule.SKU] = rule.ID
	return cloneRule(rule), nil
}

// Get returns a price rule by ID.
func (s *Store) Get(ctx context.Context, id string) (*PriceRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if id == "" {
		return nil, errors.InvalidArgument("id is required", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[id]
	if !ok {
		return nil, errors.NotFound("price not found", nil)
	}
	return cloneRule(r), nil
}

// List returns all price rules.
func (s *Store) List(ctx context.Context) ([]*PriceRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*PriceRule, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, cloneRule(r))
	}
	return out, nil
}

// Quote computes unit and total for a SKU quantity using the latest rule.
func (s *Store) Quote(ctx context.Context, sku string, qty int64) (*Quote, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if sku == "" {
		return nil, errors.InvalidArgument("sku is required", nil)
	}
	if qty <= 0 {
		return nil, errors.InvalidArgument("qty must be positive", nil)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	ruleID, ok := s.bySKU[sku]
	if !ok {
		return nil, errors.NotFound("price not found for sku", nil)
	}
	r := s.rules[ruleID]
	unit := commerce.NewMoney(r.AmountMinor, r.Currency)
	return &Quote{
		SKU:         sku,
		Qty:         qty,
		UnitMinor:   unit.Amount,
		TotalMinor:  unit.Amount * qty,
		Currency:    unit.Currency,
		PriceRuleID: r.ID,
	}, nil
}

func cloneRule(r *PriceRule) *PriceRule {
	cp := *r
	return &cp
}
