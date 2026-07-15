package tests

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	web3mem "github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/memory"
)

func TestResilientClient_GetChainID(t *testing.T) {
	inner := web3mem.NewClient(web3mem.ClientConfig{ChainID: 11155111})
	client := web3.NewResilientClient(inner, web3.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	id, err := client.GetChainID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if id.Cmp(big.NewInt(11155111)) != 0 {
		t.Fatalf("chain=%s", id)
	}
}

func TestResilientStore_AddGet(t *testing.T) {
	inner := web3mem.NewStore(web3mem.StoreConfig{})
	store := web3.NewResilientStore(inner, web3.ResilientConfig{
		CircuitBreakerEnabled: true,
		RetryEnabled:          true,
		RetryMaxAttempts:      2,
		RetryBackoff:          time.Millisecond,
	})
	cid, err := store.Add(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := store.Get(context.Background(), cid)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data=%q", data)
	}
}
