package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Ensure compile-time interface compliance.
var (
	_ payment.Provider   = (*Provider)(nil)
	_ payment.Authorizer = (*Provider)(nil)
)

// Provider implements payment.Authorizer in memory.
type Provider struct {
	transactions map[string]*payment.Transaction
	idempotency  map[string]string // key -> transaction ID
	mu           *concurrency.SmartRWMutex
}

// New creates a new memory payment provider.
func New() *Provider {
	return &Provider{
		transactions: make(map[string]*payment.Transaction),
		idempotency:  make(map[string]string),
		mu:           concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "commerce-payment-memory"}),
	}
}

func (p *Provider) Charge(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Amount.Validate(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if req.IdempotencyKey != "" {
		if existingID, ok := p.idempotency[req.IdempotencyKey]; ok {
			tx := p.transactions[existingID]
			if tx != nil && tx.Amount.Equal(req.Amount) && tx.SourceID == req.SourceID {
				cp := *tx
				return &cp, nil
			}
			return nil, payment.ErrIdempotencyConflict
		}
	}

	if req.SourceID == "fail_card" {
		return nil, payment.ErrDeclined
	}

	id := uuid.New().String()
	tx := &payment.Transaction{
		ID:             id,
		Amount:         req.Amount,
		Status:         payment.StatusSucceeded,
		SourceID:       req.SourceID,
		Description:    req.Description,
		CreatedAt:      time.Now().UTC(),
		Metadata:       cloneMetadata(req.Metadata),
		IdempotencyKey: req.IdempotencyKey,
	}
	p.transactions[id] = tx
	if req.IdempotencyKey != "" {
		p.idempotency[req.IdempotencyKey] = id
	}
	cp := *tx
	return &cp, nil
}

func (p *Provider) Authorize(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Amount.Validate(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if req.SourceID == "fail_card" {
		return nil, payment.ErrDeclined
	}

	id := uuid.New().String()
	tx := &payment.Transaction{
		ID:             id,
		Amount:         req.Amount,
		Status:         payment.StatusAuthorized,
		SourceID:       req.SourceID,
		Description:    req.Description,
		CreatedAt:      time.Now().UTC(),
		Metadata:       cloneMetadata(req.Metadata),
		IdempotencyKey: req.IdempotencyKey,
	}
	p.transactions[id] = tx
	cp := *tx
	return &cp, nil
}

func (p *Provider) Capture(ctx context.Context, req *payment.CaptureRequest) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	tx, ok := p.transactions[req.TransactionID]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}
	if tx.Status != payment.StatusAuthorized {
		return nil, payment.ErrNotAuthorized
	}

	captureAmt := req.Amount
	if captureAmt.IsZero() {
		captureAmt = tx.Amount
	} else if !captureAmt.SameCurrency(tx.Amount) {
		return nil, errors.InvalidArgument("capture currency mismatch", nil)
	} else if captureAmt.Amount > tx.Amount.Amount {
		return nil, errors.InvalidArgument("capture exceeds authorized amount", nil)
	}

	tx.Amount = captureAmt
	tx.Status = payment.StatusSucceeded
	cp := *tx
	return &cp, nil
}

func (p *Provider) Void(ctx context.Context, transactionID string) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	tx, ok := p.transactions[transactionID]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}
	if tx.Status != payment.StatusAuthorized {
		return nil, payment.ErrNotAuthorized
	}
	tx.Status = payment.StatusVoided
	cp := *tx
	return &cp, nil
}

func (p *Provider) Refund(ctx context.Context, req *payment.RefundRequest) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	tx, ok := p.transactions[req.TransactionID]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}
	if tx.Status == payment.StatusRefunded {
		return nil, errors.Conflict("transaction already refunded", nil)
	}
	if tx.Status != payment.StatusSucceeded {
		return nil, errors.InvalidArgument("only succeeded transactions can be refunded", nil)
	}

	refundAmt := req.Amount
	if refundAmt.IsZero() {
		refundAmt = tx.Amount
	} else if refundAmt.Currency == "" {
		refundAmt.Currency = tx.Amount.Currency
	}

	id := uuid.New().String()
	refundTx := &payment.Transaction{
		ID:          id,
		Amount:      refundAmt,
		Status:      payment.StatusRefunded,
		SourceID:    tx.SourceID,
		Description: fmt.Sprintf("Refund for %s: %s", tx.ID, req.Reason),
		CreatedAt:   time.Now().UTC(),
	}
	p.transactions[id] = refundTx
	tx.Status = payment.StatusRefunded
	cp := *refundTx
	return &cp, nil
}

func (p *Provider) GetTransaction(ctx context.Context, id string) (*payment.Transaction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	tx, ok := p.transactions[id]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}
	cp := *tx
	return &cp, nil
}

func (p *Provider) Close() error {
	return nil
}

func cloneMetadata(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
