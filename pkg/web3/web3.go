package web3

import (
	"context"
	"math/big"
	"time"
)

// Config holds shared Web3 package configuration.
type Config struct {
	// Driver selects a backend: "memory", "ethereum", "ipfs".
	Driver string `env:"WEB3_DRIVER" env-default:"memory"`
}

// Receipt is an SDK-free Ethereum transaction receipt summary.
type Receipt struct {
	TxHash          string
	BlockNumber     uint64
	Status          uint64
	GasUsed         uint64
	ContractAddress string
}

// Client is the primary Ethereum (EVM) client interface.
// Implementations include adapters/memory. The concrete geth wrapper under
// blockchain/ethereum is not yet adapted behind this interface (SDK isolation
// is best-effort).
type Client interface {
	// Close releases the underlying connection.
	Close()

	// GetChainID returns the configured or discovered chain ID.
	GetChainID(ctx context.Context) (*big.Int, error)

	// GetBalance returns the balance of an address in wei.
	GetBalance(ctx context.Context, address string) (*big.Int, error)

	// GetBlockNumber returns the latest block number.
	GetBlockNumber(ctx context.Context) (uint64, error)

	// GetTransactionReceipt retrieves a transaction receipt by hash.
	GetTransactionReceipt(ctx context.Context, txHash string) (*Receipt, error)

	// Transfer sends native currency from the configured signer to a recipient.
	Transfer(ctx context.Context, to string, amountWei *big.Int) (string, error)

	// CallContract executes a read-only contract call.
	CallContract(ctx context.Context, contractAddr string, data []byte) ([]byte, error)

	// EstimateGas estimates gas for a transaction.
	EstimateGas(ctx context.Context, to string, data []byte) (uint64, error)

	// WaitForTransaction waits until a transaction is mined or ctx is done.
	WaitForTransaction(ctx context.Context, txHash string) (*Receipt, error)

	// GetAddress returns the address derived from the configured private key.
	GetAddress() (string, error)
}

// Store is the primary IPFS content-addressed storage interface.
// Implementations include adapters/memory. The HTTP client under storage/ipfs
// is a concrete scaffold, not yet adapted behind this interface.
type Store interface {
	// Add uploads content and returns its CID (or memory content id).
	Add(ctx context.Context, data []byte) (string, error)

	// AddJSON marshals and uploads JSON data.
	AddJSON(ctx context.Context, data interface{}) (string, error)

	// Get retrieves content by CID.
	Get(ctx context.Context, cid string) ([]byte, error)

	// GetJSON retrieves and unmarshals JSON by CID.
	GetJSON(ctx context.Context, cid string, result interface{}) error

	// GetURL returns a gateway-style URL for a CID.
	GetURL(cid string) string

	// Pin marks content to prevent garbage collection.
	Pin(ctx context.Context, cid string) error

	// Unpin removes a pin.
	Unpin(ctx context.Context, cid string) error

	// ListPins returns all pinned CIDs.
	ListPins(ctx context.Context) ([]string, error)
}

// SIWEMessage represents a Sign-In with Ethereum message (EIP-4361 fields).
type SIWEMessage struct {
	Domain         string
	Address        string
	Statement      string
	URI            string
	Version        string
	ChainID        int
	Nonce          string
	IssuedAt       time.Time
	ExpirationTime *time.Time
	NotBefore      *time.Time
	RequestID      string
	Resources      []string
}

// Verifier verifies Sign-In with Ethereum (SIWE) messages and signatures.
// Implementations include identity.SIWEVerifier (cryptographic) and
// adapters/memory (test double).
type Verifier interface {
	// GenerateNonce creates a random nonce for SIWE.
	GenerateNonce() (string, error)

	// CreateMessage builds a new SIWE message with a fresh nonce.
	CreateMessage(domain, address, uri, statement string, chainID int) (*SIWEMessage, error)

	// Verify checks expiration, not-before, nonce reuse, and signature recovery.
	Verify(ctx context.Context, message *SIWEMessage, signature string) (bool, error)
}
