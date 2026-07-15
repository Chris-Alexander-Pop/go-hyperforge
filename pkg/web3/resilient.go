package web3

import (
	"context"
	"math/big"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/resilience"
)

// Ensure compile-time interface compliance.
var (
	_ Client = (*ResilientClient)(nil)
	_ Store  = (*ResilientStore)(nil)
)

// ResilientConfig configures resilient Web3 wrappers.
type ResilientConfig struct {
	CircuitBreakerEnabled   bool
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration

	RetryEnabled     bool
	RetryMaxAttempts int
	RetryBackoff     time.Duration
}

func newResilientParts(name string, cfg ResilientConfig) (*resilience.CircuitBreaker, resilience.RetryConfig) {
	var cb *resilience.CircuitBreaker
	var retryCfg resilience.RetryConfig

	if cfg.CircuitBreakerEnabled {
		threshold := cfg.CircuitBreakerThreshold
		if threshold <= 0 {
			threshold = 5
		}
		timeout := cfg.CircuitBreakerTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		cb = resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
			Name:             name,
			FailureThreshold: threshold,
			SuccessThreshold: 2,
			Timeout:          timeout,
		})
	}

	if cfg.RetryEnabled && cfg.RetryMaxAttempts > 0 {
		backoff := cfg.RetryBackoff
		if backoff <= 0 {
			backoff = 100 * time.Millisecond
		}
		retryCfg = resilience.RetryConfig{
			MaxAttempts:    cfg.RetryMaxAttempts,
			InitialBackoff: backoff,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2.0,
			Jitter:         0.1,
			RetryIf:        isTransientWeb3Err,
		}
	}

	return cb, retryCfg
}

func isTransientWeb3Err(err error) bool {
	if err == nil {
		return false
	}
	switch errors.Code(err) {
	case CodeNotFound, CodeInvalidConfig, CodeInvalidSignature, CodeNonceReused,
		CodeMessageExpired, CodeMessageNotYet, CodeNoSigner, CodeCanceled:
		return false
	}
	return true
}

func isExpectedWeb3Err(err error) bool {
	if err == nil {
		return false
	}
	switch errors.Code(err) {
	case CodeNotFound, CodeInvalidConfig, CodeInvalidSignature, CodeNonceReused,
		CodeMessageExpired, CodeMessageNotYet, CodeNoSigner, CodeCanceled:
		return true
	}
	return false
}

func executeResilient(ctx context.Context, cb *resilience.CircuitBreaker, retryCfg resilience.RetryConfig, fn resilience.Executor) error {
	operation := fn
	if cb != nil {
		cbFn := operation
		operation = func(ctx context.Context) error {
			var opErr error
			cbErr := cb.Execute(ctx, func(ctx context.Context) error {
				opErr = cbFn(ctx)
				if isExpectedWeb3Err(opErr) {
					return nil
				}
				return opErr
			})
			if cbErr != nil {
				return cbErr
			}
			return opErr
		}
	}
	if retryCfg.MaxAttempts > 0 {
		return resilience.Retry(ctx, retryCfg, operation)
	}
	return operation(ctx)
}

// ResilientClient wraps a Client with circuit breaker and retry.
type ResilientClient struct {
	next     Client
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientClient wraps next with resilience features.
func NewResilientClient(next Client, cfg ResilientConfig) *ResilientClient {
	cb, retryCfg := newResilientParts("web3-client", cfg)
	return &ResilientClient{next: next, cb: cb, retryCfg: retryCfg}
}

func (c *ResilientClient) execute(ctx context.Context, fn resilience.Executor) error {
	return executeResilient(ctx, c.cb, c.retryCfg, fn)
}

// Close releases the underlying connection.
func (c *ResilientClient) Close() {
	c.next.Close()
}

// GetChainID runs GetChainID with resilience.
func (c *ResilientClient) GetChainID(ctx context.Context) (*big.Int, error) {
	var id *big.Int
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		id, e = c.next.GetChainID(ctx)
		return e
	})
	return id, err
}

// GetBalance runs GetBalance with resilience.
func (c *ResilientClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	var bal *big.Int
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		bal, e = c.next.GetBalance(ctx, address)
		return e
	})
	return bal, err
}

// GetBlockNumber runs GetBlockNumber with resilience.
func (c *ResilientClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	var n uint64
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		n, e = c.next.GetBlockNumber(ctx)
		return e
	})
	return n, err
}

// GetTransactionReceipt runs GetTransactionReceipt with resilience.
func (c *ResilientClient) GetTransactionReceipt(ctx context.Context, txHash string) (*Receipt, error) {
	var r *Receipt
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		r, e = c.next.GetTransactionReceipt(ctx, txHash)
		return e
	})
	return r, err
}

// Transfer runs Transfer with resilience.
func (c *ResilientClient) Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error) {
	var hash string
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		hash, e = c.next.Transfer(ctx, to, amountWei)
		return e
	})
	return hash, err
}

// CallContract runs CallContract with resilience.
func (c *ResilientClient) CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error) {
	var out []byte
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		out, e = c.next.CallContract(ctx, contractAddr, data)
		return e
	})
	return out, err
}

// EstimateGas runs EstimateGas with resilience.
func (c *ResilientClient) EstimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	var gas uint64
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		gas, e = c.next.EstimateGas(ctx, to, data)
		return e
	})
	return gas, err
}

// WaitForTransaction runs WaitForTransaction with resilience.
func (c *ResilientClient) WaitForTransaction(ctx context.Context, txHash string) (*Receipt, error) {
	var r *Receipt
	err := c.execute(ctx, func(ctx context.Context) error {
		var e error
		r, e = c.next.WaitForTransaction(ctx, txHash)
		return e
	})
	return r, err
}

// GetAddress delegates without resilience (local config).
func (c *ResilientClient) GetAddress() (string, error) {
	return c.next.GetAddress()
}

// Unwrap returns the underlying client.
func (c *ResilientClient) Unwrap() Client {
	return c.next
}

// ResilientStore wraps a Store with circuit breaker and retry.
type ResilientStore struct {
	next     Store
	cb       *resilience.CircuitBreaker
	retryCfg resilience.RetryConfig
}

// NewResilientStore wraps next with resilience features.
func NewResilientStore(next Store, cfg ResilientConfig) *ResilientStore {
	cb, retryCfg := newResilientParts("web3-store", cfg)
	return &ResilientStore{next: next, cb: cb, retryCfg: retryCfg}
}

func (s *ResilientStore) execute(ctx context.Context, fn resilience.Executor) error {
	return executeResilient(ctx, s.cb, s.retryCfg, fn)
}

// Add runs Add with resilience.
func (s *ResilientStore) Add(ctx context.Context, data []byte) (string, error) {
	var cid string
	err := s.execute(ctx, func(ctx context.Context) error {
		var e error
		cid, e = s.next.Add(ctx, data)
		return e
	})
	return cid, err
}

// AddJSON runs AddJSON with resilience.
func (s *ResilientStore) AddJSON(ctx context.Context, data interface{}) (string, error) {
	var cid string
	err := s.execute(ctx, func(ctx context.Context) error {
		var e error
		cid, e = s.next.AddJSON(ctx, data)
		return e
	})
	return cid, err
}

// Get runs Get with resilience.
func (s *ResilientStore) Get(ctx context.Context, cid string) ([]byte, error) {
	var data []byte
	err := s.execute(ctx, func(ctx context.Context) error {
		var e error
		data, e = s.next.Get(ctx, cid)
		return e
	})
	return data, err
}

// GetJSON runs GetJSON with resilience.
func (s *ResilientStore) GetJSON(ctx context.Context, cid string, result interface{}) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.GetJSON(ctx, cid, result)
	})
}

// GetURL delegates without resilience (local formatting).
func (s *ResilientStore) GetURL(cid string) string {
	return s.next.GetURL(cid)
}

// Pin runs Pin with resilience.
func (s *ResilientStore) Pin(ctx context.Context, cid string) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.Pin(ctx, cid)
	})
}

// Unpin runs Unpin with resilience.
func (s *ResilientStore) Unpin(ctx context.Context, cid string) error {
	return s.execute(ctx, func(ctx context.Context) error {
		return s.next.Unpin(ctx, cid)
	})
}

// ListPins runs ListPins with resilience.
func (s *ResilientStore) ListPins(ctx context.Context) ([]string, error) {
	var pins []string
	err := s.execute(ctx, func(ctx context.Context) error {
		var e error
		pins, e = s.next.ListPins(ctx)
		return e
	})
	return pins, err
}

// Unwrap returns the underlying store.
func (s *ResilientStore) Unwrap() Store {
	return s.next
}
