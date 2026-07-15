package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
)

// Ensure Solana implements web3.SolanaClient.
var _ web3.SolanaClient = (*Solana)(nil)

// SolanaConfig configures the in-memory Solana client.
type SolanaConfig struct {
	Balances      map[string]uint64
	TokenBalances map[string]string
	BlockHeight   uint64
	Slot          uint64
	Blockhash     string
}

// Solana is an in-memory SolanaClient for tests.
type Solana struct {
	mu            *concurrency.SmartRWMutex
	balances      map[string]uint64
	tokenBalances map[string]string
	accounts      map[string]map[string]interface{}
	txs           map[string]map[string]interface{}
	height        atomic.Uint64
	slot          atomic.Uint64
	blockhash     string
	txSeq         atomic.Uint64
	closed        atomic.Bool
}

// NewSolana creates an in-memory Solana client.
func NewSolana(cfg SolanaConfig) *Solana {
	if cfg.BlockHeight == 0 {
		cfg.BlockHeight = 1000
	}
	if cfg.Slot == 0 {
		cfg.Slot = 2000
	}
	if cfg.Blockhash == "" {
		cfg.Blockhash = "MemoryBlockhash111111111111111111111111111"
	}
	balances := make(map[string]uint64)
	for k, v := range cfg.Balances {
		balances[k] = v
	}
	tokens := make(map[string]string)
	for k, v := range cfg.TokenBalances {
		tokens[k] = v
	}
	s := &Solana{
		mu:            concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-memory-solana"}),
		balances:      balances,
		tokenBalances: tokens,
		accounts:      make(map[string]map[string]interface{}),
		txs:           make(map[string]map[string]interface{}),
		blockhash:     cfg.Blockhash,
	}
	s.height.Store(cfg.BlockHeight)
	s.slot.Store(cfg.Slot)
	return s
}

// Close marks the client closed.
func (s *Solana) Close() { s.closed.Store(true) }

func (s *Solana) guard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.closed.Load() {
		return web3.ErrConnectionFailed(nil)
	}
	return nil
}

// GetBalance implements web3.SolanaClient.
func (s *Solana) GetBalance(ctx context.Context, address string) (uint64, error) {
	if err := s.guard(ctx); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balances[address], nil
}

// GetBlockHeight implements web3.SolanaClient.
func (s *Solana) GetBlockHeight(ctx context.Context) (uint64, error) {
	if err := s.guard(ctx); err != nil {
		return 0, err
	}
	return s.height.Load(), nil
}

// GetSlot implements web3.SolanaClient.
func (s *Solana) GetSlot(ctx context.Context) (uint64, error) {
	if err := s.guard(ctx); err != nil {
		return 0, err
	}
	return s.slot.Load(), nil
}

// GetTransaction implements web3.SolanaClient.
func (s *Solana) GetTransaction(ctx context.Context, signature string) (map[string]interface{}, error) {
	if err := s.guard(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tx, ok := s.txs[signature]
	if !ok {
		return nil, web3.ErrNotFound("transaction", nil)
	}
	cp := make(map[string]interface{}, len(tx))
	for k, v := range tx {
		cp[k] = v
	}
	return cp, nil
}

// GetAccountInfo implements web3.SolanaClient.
func (s *Solana) GetAccountInfo(ctx context.Context, address string) (map[string]interface{}, error) {
	if err := s.guard(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	info, ok := s.accounts[address]
	if !ok {
		return map[string]interface{}{"value": nil}, nil
	}
	cp := make(map[string]interface{}, len(info))
	for k, v := range info {
		cp[k] = v
	}
	return cp, nil
}

// SendTransaction implements web3.SolanaClient.
func (s *Solana) SendTransaction(ctx context.Context, signedTx string) (string, error) {
	if err := s.guard(ctx); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(signedTx))
	sig := hex.EncodeToString(sum[:16])
	s.mu.Lock()
	s.txs[sig] = map[string]interface{}{
		"signature": sig,
		"slot":      s.slot.Load(),
	}
	s.mu.Unlock()
	s.slot.Add(1)
	s.height.Add(1)
	s.txSeq.Add(1)
	return sig, nil
}

// GetRecentBlockhash implements web3.SolanaClient.
func (s *Solana) GetRecentBlockhash(ctx context.Context) (string, error) {
	if err := s.guard(ctx); err != nil {
		return "", err
	}
	return s.blockhash, nil
}

// GetTokenAccountBalance implements web3.SolanaClient.
func (s *Solana) GetTokenAccountBalance(ctx context.Context, tokenAccount string) (string, error) {
	if err := s.guard(ctx); err != nil {
		return "", err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	amt, ok := s.tokenBalances[tokenAccount]
	if !ok {
		return "0", nil
	}
	return amt, nil
}

// SetBalance seeds a lamport balance (test helper).
func (s *Solana) SetBalance(address string, lamports uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.balances[address] = lamports
}
