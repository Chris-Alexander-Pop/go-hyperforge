package web3

import (
	"context"
	"math/big"

	"github.com/chris-alexander-pop/system-design-library/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Ensure compile-time interface compliance.
var (
	_ Client   = (*InstrumentedClient)(nil)
	_ Store    = (*InstrumentedStore)(nil)
	_ Verifier = (*InstrumentedVerifier)(nil)
)

// InstrumentedClient wraps a Client with logging and OpenTelemetry spans.
type InstrumentedClient struct {
	next   Client
	tracer trace.Tracer
}

// NewInstrumentedClient decorates client with logging and tracing.
func NewInstrumentedClient(next Client) *InstrumentedClient {
	return &InstrumentedClient{
		next:   next,
		tracer: otel.Tracer("pkg/web3"),
	}
}

// Close delegates to the underlying client.
func (c *InstrumentedClient) Close() {
	logger.L().Info("closing Web3 client")
	c.next.Close()
}

// GetChainID logs and traces chain ID lookup.
func (c *InstrumentedClient) GetChainID(ctx context.Context) (*big.Int, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.GetChainID")
	defer span.End()

	id, err := c.next.GetChainID(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "get chain ID failed", "error", err)
		return nil, err
	}
	if id != nil {
		span.SetAttributes(attribute.String("web3.chain_id", id.String()))
	}
	return id, nil
}

// GetBalance logs and traces a balance query.
func (c *InstrumentedClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.GetBalance", trace.WithAttributes(
		attribute.String("web3.address", address),
	))
	defer span.End()

	bal, err := c.next.GetBalance(ctx, address)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "get balance failed", "address", address, "error", err)
		return nil, err
	}
	return bal, nil
}

// GetBlockNumber logs and traces block number lookup.
func (c *InstrumentedClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.GetBlockNumber")
	defer span.End()

	n, err := c.next.GetBlockNumber(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "get block number failed", "error", err)
		return 0, err
	}
	span.SetAttributes(attribute.Int64("web3.block_number", int64(n)))
	return n, nil
}

// GetTransactionReceipt logs and traces receipt lookup.
func (c *InstrumentedClient) GetTransactionReceipt(ctx context.Context, txHash string) (*Receipt, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.GetTransactionReceipt", trace.WithAttributes(
		attribute.String("web3.tx_hash", txHash),
	))
	defer span.End()

	receipt, err := c.next.GetTransactionReceipt(ctx, txHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "get transaction receipt failed", "tx_hash", txHash, "error", err)
		return nil, err
	}
	return receipt, nil
}

// Transfer logs and traces a transfer.
func (c *InstrumentedClient) Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error) {
	amount := "0"
	if amountWei != nil {
		amount = amountWei.String()
	}
	ctx, span := c.tracer.Start(ctx, "web3.Client.Transfer", trace.WithAttributes(
		attribute.String("web3.to", to),
		attribute.String("web3.amount_wei", amount),
	))
	defer span.End()

	logger.L().InfoContext(ctx, "transferring native currency", "to", to, "amount_wei", amount)
	txHash, err := c.next.Transfer(ctx, to, amountWei)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "transfer failed", "to", to, "error", err)
		return "", err
	}
	span.SetAttributes(attribute.String("web3.tx_hash", txHash))
	return txHash, nil
}

// CallContract logs and traces a contract call.
func (c *InstrumentedClient) CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.CallContract", trace.WithAttributes(
		attribute.String("web3.contract", contractAddr),
		attribute.Int("web3.data_size", len(data)),
	))
	defer span.End()

	result, err := c.next.CallContract(ctx, contractAddr, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "contract call failed", "contract", contractAddr, "error", err)
		return nil, err
	}
	return result, nil
}

// EstimateGas logs and traces gas estimation.
func (c *InstrumentedClient) EstimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.EstimateGas", trace.WithAttributes(
		attribute.String("web3.to", to),
	))
	defer span.End()

	gas, err := c.next.EstimateGas(ctx, to, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "gas estimation failed", "to", to, "error", err)
		return 0, err
	}
	span.SetAttributes(attribute.Int64("web3.gas", int64(gas)))
	return gas, nil
}

// WaitForTransaction logs and traces waiting for a receipt.
func (c *InstrumentedClient) WaitForTransaction(ctx context.Context, txHash string) (*Receipt, error) {
	ctx, span := c.tracer.Start(ctx, "web3.Client.WaitForTransaction", trace.WithAttributes(
		attribute.String("web3.tx_hash", txHash),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "waiting for transaction", "tx_hash", txHash)
	receipt, err := c.next.WaitForTransaction(ctx, txHash)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "wait for transaction failed", "tx_hash", txHash, "error", err)
		return nil, err
	}
	return receipt, nil
}

// GetAddress delegates to the underlying client.
func (c *InstrumentedClient) GetAddress() (string, error) {
	return c.next.GetAddress()
}

// InstrumentedStore wraps a Store with logging and OpenTelemetry spans.
type InstrumentedStore struct {
	next   Store
	tracer trace.Tracer
}

// NewInstrumentedStore decorates store with logging and tracing.
func NewInstrumentedStore(next Store) *InstrumentedStore {
	return &InstrumentedStore{
		next:   next,
		tracer: otel.Tracer("pkg/web3"),
	}
}

// Add logs and traces content upload.
func (s *InstrumentedStore) Add(ctx context.Context, data []byte) (string, error) {
	ctx, span := s.tracer.Start(ctx, "web3.Store.Add", trace.WithAttributes(
		attribute.Int("web3.data_size", len(data)),
	))
	defer span.End()

	cid, err := s.next.Add(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS add failed", "error", err)
		return "", err
	}
	span.SetAttributes(attribute.String("web3.cid", cid))
	return cid, nil
}

// AddJSON logs and traces JSON upload.
func (s *InstrumentedStore) AddJSON(ctx context.Context, data interface{}) (string, error) {
	ctx, span := s.tracer.Start(ctx, "web3.Store.AddJSON")
	defer span.End()

	cid, err := s.next.AddJSON(ctx, data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS add JSON failed", "error", err)
		return "", err
	}
	span.SetAttributes(attribute.String("web3.cid", cid))
	return cid, nil
}

// Get logs and traces content retrieval.
func (s *InstrumentedStore) Get(ctx context.Context, cid string) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "web3.Store.Get", trace.WithAttributes(
		attribute.String("web3.cid", cid),
	))
	defer span.End()

	data, err := s.next.Get(ctx, cid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS get failed", "cid", cid, "error", err)
		return nil, err
	}
	return data, nil
}

// GetJSON logs and traces JSON retrieval.
func (s *InstrumentedStore) GetJSON(ctx context.Context, cid string, result interface{}) error {
	ctx, span := s.tracer.Start(ctx, "web3.Store.GetJSON", trace.WithAttributes(
		attribute.String("web3.cid", cid),
	))
	defer span.End()

	err := s.next.GetJSON(ctx, cid, result)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS get JSON failed", "cid", cid, "error", err)
		return err
	}
	return nil
}

// GetURL delegates to the underlying store.
func (s *InstrumentedStore) GetURL(cid string) string {
	return s.next.GetURL(cid)
}

// Pin logs and traces pinning.
func (s *InstrumentedStore) Pin(ctx context.Context, cid string) error {
	ctx, span := s.tracer.Start(ctx, "web3.Store.Pin", trace.WithAttributes(
		attribute.String("web3.cid", cid),
	))
	defer span.End()

	err := s.next.Pin(ctx, cid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS pin failed", "cid", cid, "error", err)
		return err
	}
	return nil
}

// Unpin logs and traces unpinning.
func (s *InstrumentedStore) Unpin(ctx context.Context, cid string) error {
	ctx, span := s.tracer.Start(ctx, "web3.Store.Unpin", trace.WithAttributes(
		attribute.String("web3.cid", cid),
	))
	defer span.End()

	err := s.next.Unpin(ctx, cid)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS unpin failed", "cid", cid, "error", err)
		return err
	}
	return nil
}

// ListPins logs and traces pin listing.
func (s *InstrumentedStore) ListPins(ctx context.Context) ([]string, error) {
	ctx, span := s.tracer.Start(ctx, "web3.Store.ListPins")
	defer span.End()

	pins, err := s.next.ListPins(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "IPFS list pins failed", "error", err)
		return nil, err
	}
	span.SetAttributes(attribute.Int("web3.pin_count", len(pins)))
	return pins, nil
}

// InstrumentedVerifier wraps a Verifier with logging and OpenTelemetry spans.
type InstrumentedVerifier struct {
	next   Verifier
	tracer trace.Tracer
}

// NewInstrumentedVerifier decorates verifier with logging and tracing.
func NewInstrumentedVerifier(next Verifier) *InstrumentedVerifier {
	return &InstrumentedVerifier{
		next:   next,
		tracer: otel.Tracer("pkg/web3"),
	}
}

// GenerateNonce logs and traces nonce generation.
func (v *InstrumentedVerifier) GenerateNonce() (string, error) {
	nonce, err := v.next.GenerateNonce()
	if err != nil {
		logger.L().Error("SIWE nonce generation failed", "error", err)
		return "", err
	}
	return nonce, nil
}

// CreateMessage logs and traces SIWE message creation.
func (v *InstrumentedVerifier) CreateMessage(domain, address, uri, statement string, chainID int) (*SIWEMessage, error) {
	logger.L().Debug("creating SIWE message", "domain", domain, "address", address, "chain_id", chainID)
	msg, err := v.next.CreateMessage(domain, address, uri, statement, chainID)
	if err != nil {
		logger.L().Error("SIWE message creation failed", "error", err)
		return nil, err
	}
	return msg, nil
}

// Verify logs and traces SIWE verification.
func (v *InstrumentedVerifier) Verify(ctx context.Context, message *SIWEMessage, signature string) (bool, error) {
	addr := ""
	nonce := ""
	if message != nil {
		addr = message.Address
		nonce = message.Nonce
	}
	ctx, span := v.tracer.Start(ctx, "web3.Verifier.Verify", trace.WithAttributes(
		attribute.String("web3.address", addr),
		attribute.String("web3.nonce", nonce),
	))
	defer span.End()

	logger.L().DebugContext(ctx, "verifying SIWE signature", "address", addr)
	ok, err := v.next.Verify(ctx, message, signature)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.L().ErrorContext(ctx, "SIWE verify failed", "address", addr, "error", err)
		return false, err
	}
	span.SetAttributes(attribute.Bool("web3.valid", ok))
	return ok, nil
}
