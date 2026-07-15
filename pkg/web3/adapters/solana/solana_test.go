package solana_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/solana"
	"github.com/stretchr/testify/require"
)

func TestSolanaClientHTTPRPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		switch method {
		case "getBalance":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]uint64{"value": 42},
			})
		case "getBlockHeight":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1, "result": 100,
			})
		case "getSlot":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1, "result": 200,
			})
		case "getLatestBlockhash":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{
					"value": map[string]string{"blockhash": "hash123"},
				},
			})
		case "sendTransaction":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1, "result": "sigABC",
			})
		case "getTokenAccountBalance":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{
					"value": map[string]string{"amount": "999"},
				},
			})
		case "getAccountInfo":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{"value": map[string]string{"owner": "sys"}},
			})
		case "getTransaction":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{"slot": 1.0},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"error": map[string]interface{}{"code": -32601, "message": "unknown"},
			})
		}
	}))
	defer srv.Close()

	c, err := solana.New(solana.Config{RPCURL: srv.URL})
	require.NoError(t, err)
	defer c.Close()

	var _ web3.SolanaClient = c
	ctx := context.Background()

	bal, err := c.GetBalance(ctx, "Addr")
	require.NoError(t, err)
	require.Equal(t, uint64(42), bal)

	h, err := c.GetBlockHeight(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(100), h)

	slot, err := c.GetSlot(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(200), slot)

	hash, err := c.GetRecentBlockhash(ctx)
	require.NoError(t, err)
	require.Equal(t, "hash123", hash)

	sig, err := c.SendTransaction(ctx, "base64tx")
	require.NoError(t, err)
	require.Equal(t, "sigABC", sig)

	amt, err := c.GetTokenAccountBalance(ctx, "tok")
	require.NoError(t, err)
	require.Equal(t, "999", amt)

	info, err := c.GetAccountInfo(ctx, "acc")
	require.NoError(t, err)
	require.NotNil(t, info)

	tx, err := c.GetTransaction(ctx, "sig")
	require.NoError(t, err)
	require.NotNil(t, tx)
}
