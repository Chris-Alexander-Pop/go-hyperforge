package ethereum

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/geth"
)

// Config is an alias for geth.Config.
type Config = geth.Config

// Client is an alias for geth.Client (implements web3.Client).
type Client = geth.Client

// New creates a geth-backed Ethereum client.
func New(cfg Config) (*Client, error) {
	return geth.New(cfg)
}
