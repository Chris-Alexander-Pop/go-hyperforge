// Package solana is a thin re-export of adapters/solana (web3.SolanaClient).
package solana

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/solana"
)

// Config is an alias for adapters/solana.Config.
type Config = solana.Config

// Client is an alias for adapters/solana.Client (implements web3.SolanaClient).
type Client = solana.Client

// New creates a Solana JSON-RPC client.
func New(cfg Config) (*Client, error) {
	return solana.New(cfg)
}
