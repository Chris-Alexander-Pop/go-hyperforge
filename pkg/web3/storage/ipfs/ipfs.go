package ipfs

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/adapters/kubo"
)

// Config is an alias for kubo.Config.
type Config = kubo.Config

// Client is an alias for kubo.Store (implements web3.Store).
// Named Client for backward compatibility with the previous HTTP scaffold.
type Client = kubo.Store

// New creates a Kubo/IPFS-backed store.
func New(cfg Config) (*Client, error) {
	return kubo.New(cfg)
}
