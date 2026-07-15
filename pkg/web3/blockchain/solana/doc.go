/*
Package solana provides a JSON-RPC Solana client scaffold.

This package talks HTTP JSON-RPC directly (no official Solana SDK). It is not
yet behind a root pkg/web3 interface. Prefer defining consumers against interfaces
and using adapters/memory for Ethereum/IPFS/SIWE until a Solana root interface lands.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/web3/blockchain/solana"

	client, err := solana.New(solana.Config{RPCURL: "https://api.mainnet-beta.solana.com"})
	balance, err := client.GetBalance(ctx, "...")
*/
package solana
