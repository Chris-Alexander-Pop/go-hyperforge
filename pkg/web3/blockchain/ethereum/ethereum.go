package ethereum

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Config holds Ethereum client configuration.
type Config struct {
	// RPCURL is the JSON-RPC endpoint
	RPCURL string

	// PrivateKey for signing transactions (optional)
	PrivateKey string

	// ChainID for the network
	ChainID int64
}

// Client wraps go-ethereum for simplified blockchain access.
type Client struct {
	eth     *ethclient.Client
	config  Config
	chainID *big.Int
	signer  *ecdsa.PrivateKey
}

// New creates a new Ethereum client.
func New(cfg Config) (*Client, error) {
	eth, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, pkgerrors.Internal("failed to connect to Ethereum node", err)
	}

	client := &Client{
		eth:    eth,
		config: cfg,
	}

	if cfg.ChainID > 0 {
		client.chainID = big.NewInt(cfg.ChainID)
	}

	if cfg.PrivateKey != "" {
		key, err := crypto.HexToECDSA(cfg.PrivateKey)
		if err != nil {
			return nil, pkgerrors.InvalidArgument("invalid private key", err)
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
		return c.chainID, nil
	}

	chainID, err := c.eth.ChainID(ctx)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get chain ID", err)
	}
	c.chainID = chainID
	return chainID, nil
}

// GetBalance returns the balance of an address in wei.
func (c *Client) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	addr := common.HexToAddress(address)
	balance, err := c.eth.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get balance", err)
	}
	return balance, nil
}

// GetBlockNumber returns the latest block number.
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	blockNum, err := c.eth.BlockNumber(ctx)
	if err != nil {
		return 0, pkgerrors.Internal("failed to get block number", err)
	}
	return blockNum, nil
}

// GetTransactionReceipt retrieves a transaction receipt.
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	receipt, err := c.eth.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, pkgerrors.NotFound("transaction not found", err)
	}
	return receipt, nil
}

// SendTransaction sends a signed transaction.
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) (string, error) {
	err := c.eth.SendTransaction(ctx, tx)
	if err != nil {
		return "", pkgerrors.Internal("failed to send transaction", err)
	}
	return tx.Hash().Hex(), nil
}

// Transfer sends ETH from the configured signer to a recipient.
func (c *Client) Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error) {
	if c.signer == nil {
		return "", pkgerrors.InvalidArgument("no signer configured", nil)
	}

	fromAddress := crypto.PubkeyToAddress(c.signer.PublicKey)
	toAddress := common.HexToAddress(to)

	nonce, err := c.eth.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", pkgerrors.Internal("failed to get nonce", err)
	}

	gasPrice, err := c.eth.SuggestGasPrice(ctx)
	if err != nil {
		return "", pkgerrors.Internal("failed to get gas price", err)
	}

	gasLimit := uint64(21000) // Standard transfer

	chainID, err := c.GetChainID(ctx)
	if err != nil {
		return "", err
	}

	tx := types.NewTransaction(nonce, toAddress, amountWei, gasLimit, gasPrice, nil)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.signer)
	if err != nil {
		return "", pkgerrors.Internal("failed to sign transaction", err)
	}

	return c.SendTransaction(ctx, signedTx)
}

// CallContract executes a contract call (read-only).
func (c *Client) CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error) {
	addr := common.HexToAddress(contractAddr)
	msg := ethereum.CallMsg{
		To:   &addr,
		Data: data,
	}

	result, err := c.eth.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, pkgerrors.Internal("contract call failed", err)
	}
	return result, nil
}

// EstimateGas estimates gas for a transaction.
func (c *Client) EstimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	toAddr := common.HexToAddress(to)
	msg := ethereum.CallMsg{
		To:   &toAddr,
		Data: data,
	}

	gas, err := c.eth.EstimateGas(ctx, msg)
	if err != nil {
		return 0, pkgerrors.Internal("gas estimation failed", err)
	}
	return gas, nil
}

// WaitForTransaction waits for a transaction to be mined.
func (c *Client) WaitForTransaction(ctx context.Context, txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)

	for {
		receipt, err := c.eth.TransactionReceipt(ctx, hash)
		if err == nil {
			return receipt, nil
		}

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, pkgerrors.New(pkgerrors.CodeDeadlineExceeded, "timed out waiting for transaction", ctx.Err())
			}
			return nil, pkgerrors.New(pkgerrors.CodeCanceled, "context canceled while waiting for transaction", ctx.Err())
		default:
			// Continue polling
		}
	}
}

// GetAddress returns the address derived from the configured private key.
func (c *Client) GetAddress() (string, error) {
	if c.signer == nil {
		return "", pkgerrors.InvalidArgument("no signer configured", nil)
	}
	return crypto.PubkeyToAddress(c.signer.PublicKey).Hex(), nil
}
