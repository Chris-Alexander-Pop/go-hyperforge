// Package geth implements web3.Client using go-ethereum ethclient.
//
// Prefer this adapter (or adapters/memory) over importing blockchain/ethereum
// directly; the ethereum package is a thin wrapper around this adapter.
package geth

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Ensure compile-time interface compliance.
var _ web3.Client = (*Client)(nil)

// Config holds Ethereum / geth client configuration.
type Config struct {
	// RPCURL is the JSON-RPC endpoint.
	RPCURL string

	// PrivateKey for signing transactions (optional hex without 0x).
	PrivateKey string

	// ChainID for the network (0 discovers via RPC).
	ChainID int64
}

// Client implements web3.Client via go-ethereum ethclient.
type Client struct {
	eth     *ethclient.Client
	config  Config
	chainID *big.Int
	signer  *ecdsa.PrivateKey
}

// New creates a geth-backed web3.Client.
func New(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		return nil, web3.ErrInvalidConfig("RPC URL is required", nil)
	}
	eth, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, web3.ErrConnectionFailed(err)
	}

	client := &Client{eth: eth, config: cfg}
	if cfg.ChainID > 0 {
		client.chainID = big.NewInt(cfg.ChainID)
	}
	if cfg.PrivateKey != "" {
		key, err := crypto.HexToECDSA(cfg.PrivateKey)
		if err != nil {
			eth.Close()
			return nil, web3.ErrInvalidConfig("invalid private key", err)
		}
		client.signer = key
	}
	return client, nil
}

// Close closes the client connection.
func (c *Client) Close() {
	if c.eth != nil {
		c.eth.Close()
	}
}

// GetChainID returns the chain ID.
func (c *Client) GetChainID(ctx context.Context) (*big.Int, error) {
	if c.chainID != nil {
		return new(big.Int).Set(c.chainID), nil
	}
	chainID, err := c.eth.ChainID(ctx)
	if err != nil {
		return nil, web3.ErrRPCFailed("ChainID", err)
	}
	c.chainID = chainID
	return new(big.Int).Set(chainID), nil
}

// GetBalance returns the balance of an address in wei.
func (c *Client) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	balance, err := c.eth.BalanceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return nil, web3.ErrRPCFailed("BalanceAt", err)
	}
	return balance, nil
}

// GetBlockNumber returns the latest block number.
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	blockNum, err := c.eth.BlockNumber(ctx)
	if err != nil {
		return 0, web3.ErrRPCFailed("BlockNumber", err)
	}
	return blockNum, nil
}

// GetTransactionReceipt retrieves a transaction receipt by hash.
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*web3.Receipt, error) {
	receipt, err := c.eth.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, web3.ErrNotFound("transaction", err)
	}
	return toReceipt(receipt), nil
}

// Transfer sends native currency from the configured signer.
func (c *Client) Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error) {
	if c.signer == nil {
		return "", web3.ErrNoSigner()
	}
	fromAddress := crypto.PubkeyToAddress(c.signer.PublicKey)
	toAddress := common.HexToAddress(to)

	nonce, err := c.eth.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", web3.ErrRPCFailed("PendingNonceAt", err)
	}
	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return "", web3.ErrRPCFailed("SuggestGasPrice", err)
	}
	chainID, err := c.GetChainID(ctx)
	if err != nil {
		return "", err
	}
	tx := types.NewTransaction(nonce, toAddress, amountWei, 21000, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.signer)
	if err != nil {
		return "", web3.ErrRPCFailed("SignTx", err)
	}
	if err := c.eth.SendTransaction(ctx, signedTx); err != nil {
		return "", web3.ErrRPCFailed("SendTransaction", err)
	}
	return signedTx.Hash().Hex(), nil
}

// CallContract executes a read-only contract call.
func (c *Client) CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error) {
	addr := common.HexToAddress(contractAddr)
	msg := ethereum.CallMsg{To: &addr, Data: data}
	result, err := c.eth.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, web3.ErrRPCFailed("CallContract", err)
	}
	return result, nil
}

// EstimateGas estimates gas for a transaction.
func (c *Client) EstimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	toAddr := common.HexToAddress(to)
	msg := ethereum.CallMsg{To: &toAddr, Data: data}
	gas, err := c.eth.EstimateGas(ctx, msg)
	if err != nil {
		return 0, web3.ErrRPCFailed("EstimateGas", err)
	}
	return gas, nil
}

// WaitForTransaction waits until a transaction is mined or ctx is done.
func (c *Client) WaitForTransaction(ctx context.Context, txHash string) (*web3.Receipt, error) {
	hash := common.HexToHash(txHash)
	for {
		receipt, err := c.eth.TransactionReceipt(ctx, hash)
		if err == nil {
			return toReceipt(receipt), nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, web3.ErrTimeout("WaitForTransaction", ctx.Err())
			}
			return nil, web3.ErrCanceled("WaitForTransaction", ctx.Err())
		default:
		}
	}
}

// GetAddress returns the address derived from the configured private key.
func (c *Client) GetAddress() (string, error) {
	if c.signer == nil {
		return "", web3.ErrNoSigner()
	}
	return crypto.PubkeyToAddress(c.signer.PublicKey).Hex(), nil
}

func toReceipt(r *types.Receipt) *web3.Receipt {
	if r == nil {
		return nil
	}
	out := &web3.Receipt{
		TxHash:  r.TxHash.Hex(),
		Status:  r.Status,
		GasUsed: r.GasUsed,
	}
	if r.BlockNumber != nil {
		out.BlockNumber = r.BlockNumber.Uint64()
	}
	if r.ContractAddress != (common.Address{}) {
		out.ContractAddress = r.ContractAddress.Hex()
	}
	return out
}
