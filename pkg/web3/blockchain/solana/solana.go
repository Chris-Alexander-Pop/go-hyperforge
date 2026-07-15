package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds Solana client configuration.
type Config struct {
	// RPCURL is the Solana RPC endpoint
	RPCURL string

	// Commitment level (processed, confirmed, finalized)
	Commitment string
}

// Client provides Solana RPC access.
type Client struct {
	rpcURL     string
	httpClient *http.Client
	commitment string
}

// New creates a new Solana client.
func New(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		cfg.RPCURL = "https://api.mainnet-beta.solana.com"
	}
	if cfg.Commitment == "" {
		cfg.Commitment = "confirmed"
	}

	return &Client{
		rpcURL:     cfg.RPCURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		commitment: cfg.Commitment,
	}, nil
}

type rpcRequest struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	Jsonrpc string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *Client) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	req := rpcRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to marshal RPC request", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, pkgerrors.Internal("failed to create RPC request", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, pkgerrors.Internal("RPC request failed", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.Internal("failed to read RPC response", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, pkgerrors.Internal("failed to parse RPC response", err)
	}

	if rpcResp.Error != nil {
		return nil, pkgerrors.Internal(fmt.Sprintf("RPC error: %s", rpcResp.Error.Message), nil)
	}

	return rpcResp.Result, nil
}

// GetBalance returns the SOL balance in lamports.
func (c *Client) GetBalance(ctx context.Context, address string) (uint64, error) {
	params := []interface{}{
		address,
		map[string]string{"commitment": c.commitment},
	}

	result, err := c.call(ctx, "getBalance", params)
	if err != nil {
		return 0, err
	}

	var resp struct {
		Value uint64 `json:"value"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return 0, pkgerrors.Internal("failed to parse balance", err)
	}

	return resp.Value, nil
}

// GetBlockHeight returns the current block height.
func (c *Client) GetBlockHeight(ctx context.Context) (uint64, error) {
	result, err := c.call(ctx, "getBlockHeight", nil)
	if err != nil {
		return 0, err
	}

	var height uint64
	if err := json.Unmarshal(result, &height); err != nil {
		return 0, pkgerrors.Internal("failed to parse block height", err)
	}

	return height, nil
}

// GetSlot returns the current slot.
func (c *Client) GetSlot(ctx context.Context) (uint64, error) {
	result, err := c.call(ctx, "getSlot", nil)
	if err != nil {
		return 0, err
	}

	var slot uint64
	if err := json.Unmarshal(result, &slot); err != nil {
		return 0, pkgerrors.Internal("failed to parse slot", err)
	}

	return slot, nil
}

// GetTransaction retrieves transaction details.
func (c *Client) GetTransaction(ctx context.Context, signature string) (map[string]interface{}, error) {
	params := []interface{}{
		signature,
		map[string]string{"encoding": "json", "commitment": c.commitment},
	}

	result, err := c.call(ctx, "getTransaction", params)
	if err != nil {
		return nil, err
	}

	var tx map[string]interface{}
	if err := json.Unmarshal(result, &tx); err != nil {
		return nil, pkgerrors.Internal("failed to parse transaction", err)
	}

	return tx, nil
}

// GetAccountInfo retrieves account data.
func (c *Client) GetAccountInfo(ctx context.Context, address string) (map[string]interface{}, error) {
	params := []interface{}{
		address,
		map[string]string{"encoding": "jsonParsed", "commitment": c.commitment},
	}

	result, err := c.call(ctx, "getAccountInfo", params)
	if err != nil {
		return nil, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(result, &info); err != nil {
		return nil, pkgerrors.Internal("failed to parse account info", err)
	}

	return info, nil
}

// SendTransaction sends a signed transaction.
func (c *Client) SendTransaction(ctx context.Context, signedTx string) (string, error) {
	params := []interface{}{
		signedTx,
		map[string]string{"encoding": "base64"},
	}

	result, err := c.call(ctx, "sendTransaction", params)
	if err != nil {
		return "", err
	}

	var signature string
	if err := json.Unmarshal(result, &signature); err != nil {
		return "", pkgerrors.Internal("failed to parse signature", err)
	}

	return signature, nil
}

// GetRecentBlockhash retrieves a recent blockhash for transactions.
func (c *Client) GetRecentBlockhash(ctx context.Context) (string, error) {
	// Use getLatestBlockhash for newer API
	result, err := c.call(ctx, "getLatestBlockhash", []interface{}{
		map[string]string{"commitment": c.commitment},
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Value struct {
			Blockhash string `json:"blockhash"`
		} `json:"value"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", pkgerrors.Internal("failed to parse blockhash", err)
	}

	return resp.Value.Blockhash, nil
}

// GetTokenAccountBalance returns SPL token balance.
func (c *Client) GetTokenAccountBalance(ctx context.Context, tokenAccount string) (string, error) {
	params := []interface{}{
		tokenAccount,
		map[string]string{"commitment": c.commitment},
	}

	result, err := c.call(ctx, "getTokenAccountBalance", params)
	if err != nil {
		return "", err
	}

	var resp struct {
		Value struct {
			Amount string `json:"amount"`
		} `json:"value"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return "", pkgerrors.Internal("failed to parse token balance", err)
	}

	return resp.Value.Amount, nil
}
