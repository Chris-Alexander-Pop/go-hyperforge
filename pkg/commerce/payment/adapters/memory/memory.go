package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/commerce/payment"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/google/uuid"
)

// Provider implements payment.Provider in memory.
type Provider struct {
	transactions map[string]*payment.Transaction
	mu           sync.RWMutex
}

// New creates a new memory provider.
func New() *Provider {
	return &Provider{
		transactions: make(map[string]*payment.Transaction),
	}
}

func (p *Provider) Charge(ctx context.Context, req *payment.ChargeRequest) (*payment.Transaction, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simulate failures
	if req.SourceID == "fail_card" {
		return nil, payment.ErrDeclined
	}

	id := uuid.New().String()
	tx := &payment.Transaction{
		ID:          id,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Status:      payment.StatusSucceeded,
		SourceID:    req.SourceID,
		Description: req.Description,
		CreatedAt:   time.Now(),
		Metadata:    req.Metadata,
	}

	p.transactions[id] = tx
	return tx, nil
}

func (p *Provider) Refund(ctx context.Context, req *payment.RefundRequest) (*payment.Transaction, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tx, ok := p.transactions[req.TransactionID]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}

	// Basic check
	if tx.Status == payment.StatusRefunded {
		return nil, errors.Conflict("transaction already refunded", nil)
	}

	// In memory simple refund logic: update status
	tx.Status = payment.StatusRefunded
	// We might store refund as separate transaction but for now just update state or return new record?
	// Real providers often return a Refund object which is distinct.
	// We'll update the tx for this simple mock.

	// Better: create a refund transaction or update existing.
	// Standards say "Refund(...) returns *Transaction". Usually that's the REFUND transaction.

	id := uuid.New().String()
	refundTx := &payment.Transaction{
		ID:          id,
		Amount:      req.Amount, // Could be partial
		Currency:    tx.Currency,
		Status:      payment.StatusRefunded,
		SourceID:    tx.SourceID,
		Description: fmt.Sprintf("Refund for %s: %s", tx.ID, req.Reason),
		CreatedAt:   time.Now(),
	}
	p.transactions[id] = refundTx

	return refundTx, nil
}

func (p *Provider) GetTransaction(ctx context.Context, id string) (*payment.Transaction, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	tx, ok := p.transactions[id]
	if !ok {
		return nil, errors.NotFound("transaction not found", nil)
	}
	return tx, nil
}

func (p *Provider) Close() error {
	return nil
}
