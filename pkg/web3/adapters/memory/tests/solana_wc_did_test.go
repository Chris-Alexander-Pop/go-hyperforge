package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/memory"
	idmem "github.com/chris-alexander-pop/go-hyperforge/pkg/web3/identity/adapters/memory"
	"github.com/stretchr/testify/require"
)

func TestMemorySolana(t *testing.T) {
	c := memory.NewSolana(memory.SolanaConfig{
		Balances:      map[string]uint64{"A": 10},
		TokenBalances: map[string]string{"T": "5"},
	})
	defer c.Close()
	ctx := context.Background()

	bal, err := c.GetBalance(ctx, "A")
	require.NoError(t, err)
	require.Equal(t, uint64(10), bal)

	sig, err := c.SendTransaction(ctx, "txbytes")
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	tx, err := c.GetTransaction(ctx, sig)
	require.NoError(t, err)
	require.Equal(t, sig, tx["signature"])

	hash, err := c.GetRecentBlockhash(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	amt, err := c.GetTokenAccountBalance(ctx, "T")
	require.NoError(t, err)
	require.Equal(t, "5", amt)
}

func TestWalletConnectSession(t *testing.T) {
	wc := memory.NewWalletConnect()
	ctx := context.Background()

	s, err := wc.Pair(ctx, "wc:topic@2?relay=test")
	require.NoError(t, err)
	require.Equal(t, web3.WCStatusPending, s.Status)

	require.NoError(t, wc.Approve(ctx, s.Topic, []string{"0xabc"}))
	got, err := wc.GetSession(ctx, s.Topic)
	require.NoError(t, err)
	require.Equal(t, web3.WCStatusApproved, got.Status)
	require.Equal(t, []string{"0xabc"}, got.Accounts)

	resp, err := wc.Request(ctx, s.Topic, "eth_accounts", []byte("[]"))
	require.NoError(t, err)
	require.Equal(t, "[]", string(resp.Result))

	require.NoError(t, wc.Disconnect(ctx, s.Topic))
	got, err = wc.GetSession(ctx, s.Topic)
	require.NoError(t, err)
	require.Equal(t, web3.WCStatusClosed, got.Status)
}

func TestDIDResolvers(t *testing.T) {
	ethr := idmem.NewEthrResolver(nil)
	web := idmem.NewWebResolver(map[string]*web3.DIDDocument{
		"did:web:example.com": {ID: "did:web:example.com", AlsoKnownAs: []string{"https://example.com"}},
	})
	reg := idmem.NewRegistry(ethr, web)
	ctx := context.Background()

	doc, err := reg.Resolve(ctx, "did:ethr:0xABCDEF")
	require.NoError(t, err)
	require.Equal(t, "did:ethr:0xABCDEF", doc.ID)
	require.NotEmpty(t, doc.VerificationMethod)

	doc, err = reg.Resolve(ctx, "did:web:example.com")
	require.NoError(t, err)
	require.Equal(t, "https://example.com", doc.AlsoKnownAs[0])

	_, err = reg.Resolve(ctx, "did:key:z6Mk")
	require.Error(t, err)
}
