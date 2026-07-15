package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync/atomic"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
)

// Ensure compile-time interface compliance.
var _ web3.Client = (*Client)(nil)

// ClientConfig configures the in-memory Ethereum client.
type ClientConfig struct {
	// ChainID defaults to 1 (mainnet) when zero.
	ChainID int64
	// Address is the signer address returned by GetAddress. Defaults to a fixed test address.
	Address string
	// InitialBalances seeds address → wei balances.
	InitialBalances map[string]*big.Int
	// StartBlock is the initial block number. Defaults to 100.
	StartBlock uint64
}

// Client is an in-memory EVM client for tests and local development.
type Client struct {
	mu        *concurrency.SmartRWMutex
	chainID   *big.Int
	address   string
	balances  map[string]*big.Int
	receipts  map[string]*web3.Receipt
	contracts map[string][]byte // contractAddr → last call result stub
	block     atomic.Uint64
	txSeq     atomic.Uint64
	closed    atomic.Bool
}

// NewClient creates an in-memory Ethereum client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.ChainID == 0 {
		cfg.ChainID = 1
	}
	if cfg.Address == "" {
		cfg.Address = "0x1111111111111111111111111111111111111111"
	}
	if cfg.StartBlock == 0 {
		cfg.StartBlock = 100
	}
	balances := make(map[string]*big.Int)
	for addr, bal := range cfg.InitialBalances {
		if bal != nil {
			balances[normalizeAddr(addr)] = new(big.Int).Set(bal)
		}
	}
	c := &Client{
		mu:        concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-memory-client"}),
		chainID:   big.NewInt(cfg.ChainID),
		address:   cfg.Address,
		balances:  balances,
		receipts:  make(map[string]*web3.Receipt),
		contracts: make(map[string][]byte),
	}
	c.block.Store(cfg.StartBlock)
	return c
}

// Close marks the client closed.
func (c *Client) Close() {
	c.closed.Store(true)
}

func (c *Client) guard(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if c.closed.Load() {
		return web3.ErrConnectionFailed(nil)
	}
	return nil
}

// GetChainID returns the configured chain ID.
func (c *Client) GetChainID(ctx context.Context) (*big.Int, error) {
	if err := c.guard(ctx); err != nil {
		return nil, err
	}
	return new(big.Int).Set(c.chainID), nil
}

// GetBalance returns the balance of an address in wei (0 if unset).
func (c *Client) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	if err := c.guard(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	bal, ok := c.balances[normalizeAddr(address)]
	if !ok {
		return big.NewInt(0), nil
	}
	return new(big.Int).Set(bal), nil
}

// GetBlockNumber returns the current simulated block number.
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	if err := c.guard(ctx); err != nil {
		return 0, err
	}
	return c.block.Load(), nil
}

// GetTransactionReceipt returns a stored receipt or not-found.
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*web3.Receipt, error) {
	if err := c.guard(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.receipts[txHash]
	if !ok {
		return nil, web3.ErrNotFound("transaction", nil)
	}
	return cloneReceipt(r), nil
}

// Transfer moves balance from the configured signer to to and records a receipt.
func (c *Client) Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error) {
	if err := c.guard(ctx); err != nil {
		return "", err
	}
	if amountWei == nil || amountWei.Sign() < 0 {
		return "", web3.ErrInvalidConfig("amount must be non-negative", nil)
	}
	if to == "" {
		return "", web3.ErrInvalidConfig("recipient address is required", nil)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	from := normalizeAddr(c.address)
	toNorm := normalizeAddr(to)
	fromBal, ok := c.balances[from]
	if !ok {
		fromBal = big.NewInt(0)
		c.balances[from] = fromBal
	}
	if fromBal.Cmp(amountWei) < 0 {
		return "", web3.ErrRPCFailed("transfer", errors.FailedPrecondition("insufficient funds", nil))
	}
	fromBal.Sub(fromBal, amountWei)
	toBal, ok := c.balances[toNorm]
	if !ok {
		toBal = big.NewInt(0)
		c.balances[toNorm] = toBal
	}
	toBal.Add(toBal, amountWei)

	n := c.block.Add(1)
	seq := c.txSeq.Add(1)
	txHash := fmt.Sprintf("0xmem%064x", seq)
	receipt := &web3.Receipt{
		TxHash:      txHash,
		BlockNumber: n,
		Status:      1,
		GasUsed:     21000,
	}
	c.receipts[txHash] = receipt
	return txHash, nil
}

// CallContract returns a stubbed response for a contract address.
// Use SetContractResponse to seed return data in tests.
func (c *Client) CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error) {
	if err := c.guard(ctx); err != nil {
		return nil, err
	}
	_ = data
	c.mu.RLock()
	defer c.mu.RUnlock()
	if out, ok := c.contracts[normalizeAddr(contractAddr)]; ok {
		return append([]byte(nil), out...), nil
	}
	return []byte{}, nil
}

// SetContractResponse seeds CallContract return data for tests.
func (c *Client) SetContractResponse(contractAddr string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contracts[normalizeAddr(contractAddr)] = append([]byte(nil), data...)
}

// EstimateGas returns a fixed estimate of 21000 for empty data, else 100000.
func (c *Client) EstimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	if err := c.guard(ctx); err != nil {
		return 0, err
	}
	_ = to
	if len(data) == 0 {
		return 21000, nil
	}
	return 100000, nil
}

// WaitForTransaction returns an existing receipt or waits until ctx is done.
func (c *Client) WaitForTransaction(ctx context.Context, txHash string) (*web3.Receipt, error) {
	if err := c.guard(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	r, ok := c.receipts[txHash]
	c.mu.RUnlock()
	if ok {
		return cloneReceipt(r), nil
	}
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, web3.ErrTimeout("WaitForTransaction", ctx.Err())
	}
	return nil, web3.ErrCanceled("WaitForTransaction", ctx.Err())
}

// GetAddress returns the configured signer address.
func (c *Client) GetAddress() (string, error) {
	if c.closed.Load() {
		return "", web3.ErrConnectionFailed(nil)
	}
	if c.address == "" {
		return "", web3.ErrNoSigner()
	}
	return c.address, nil
}

// SetBalance sets an address balance for tests.
func (c *Client) SetBalance(address string, amount *big.Int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if amount == nil {
		c.balances[normalizeAddr(address)] = big.NewInt(0)
		return
	}
	c.balances[normalizeAddr(address)] = new(big.Int).Set(amount)
}

func normalizeAddr(addr string) string {
	return strings.ToLower(addr)
}

func cloneReceipt(r *web3.Receipt) *web3.Receipt {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}

// ContentID returns a deterministic fake CID for data (exported for tests).
func ContentID(data []byte) string {
	sum := sha256.Sum256(data)
	return "bafy" + hex.EncodeToString(sum[:16])
}
