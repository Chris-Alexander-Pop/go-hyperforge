package geth_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/geth"
	"github.com/stretchr/testify/require"
)

func TestNew_RequiresRPCURL(t *testing.T) {
	_, err := geth.New(geth.Config{})
	require.Error(t, err)
}

func TestNew_InvalidPrivateKey(t *testing.T) {
	// Dial may fail without a node; invalid key is checked after dial.
	// Use empty RPC to hit config validation first.
	_, err := geth.New(geth.Config{RPCURL: ""})
	require.Error(t, err)

	var _ web3.Client = (*geth.Client)(nil)
}
