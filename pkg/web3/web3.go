package web3

import (
	"context"
	"math/big"
	"time"
)

// Config holds shared Web3 package configuration.
type Config struct {
	// Driver selects a backend: "memory", "ethereum", "solana", "ipfs".
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
// Implementations include adapters/memory and adapters/geth
// (blockchain/ethereum is a thin re-export of geth).
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
// Implementations include adapters/memory and adapters/kubo
// (storage/ipfs is a thin re-export of kubo).
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

// SolanaClient is the primary Solana JSON-RPC client interface.
// Implementations include adapters/memory (Solana) and adapters/solana
// (blockchain/solana is a thin re-export of adapters/solana).
type SolanaClient interface {
	// Close releases resources (no-op for HTTP clients).
	Close()

	// GetBalance returns the SOL balance in lamports.
	GetBalance(ctx context.Context, address string) (uint64, error)

	// GetBlockHeight returns the current block height.
	GetBlockHeight(ctx context.Context) (uint64, error)

	// GetSlot returns the current slot.
	GetSlot(ctx context.Context) (uint64, error)

	// GetTransaction retrieves transaction details by signature.
	GetTransaction(ctx context.Context, signature string) (map[string]interface{}, error)

	// GetAccountInfo retrieves account data.
	GetAccountInfo(ctx context.Context, address string) (map[string]interface{}, error)

	// SendTransaction sends a signed transaction (base64) and returns the signature.
	SendTransaction(ctx context.Context, signedTx string) (string, error)

	// GetRecentBlockhash retrieves a recent blockhash for transactions.
	GetRecentBlockhash(ctx context.Context) (string, error)

	// GetTokenAccountBalance returns an SPL token balance amount string.
	GetTokenAccountBalance(ctx context.Context, tokenAccount string) (string, error)
}

// WalletConnectSession is a thin WalletConnect session stub (no relay protocol).
// Implementations include adapters/memory.
type WalletConnectSession interface {
	// Pair creates or resumes a pairing topic with a peer URI.
	Pair(ctx context.Context, uri string) (*WCSession, error)

	// Approve marks a session approved for the given topic.
	Approve(ctx context.Context, topic string, accounts []string) error

	// Reject marks a session rejected.
	Reject(ctx context.Context, topic string, reason string) error

	// GetSession returns session state by topic.
	GetSession(ctx context.Context, topic string) (*WCSession, error)

	// Disconnect ends a session.
	Disconnect(ctx context.Context, topic string) error

	// Request records a JSON-RPC style request against an active session.
	Request(ctx context.Context, topic, method string, params []byte) (*WCResponse, error)
}

// WCSession is WalletConnect session state (stub).
type WCSession struct {
	Topic    string
	URI      string
	Accounts []string
	Status   WCSessionStatus
	Metadata map[string]string
}

// WCSessionStatus is the lifecycle state of a WalletConnect session.
type WCSessionStatus string

const (
	WCStatusPending  WCSessionStatus = "pending"
	WCStatusApproved WCSessionStatus = "approved"
	WCStatusRejected WCSessionStatus = "rejected"
	WCStatusClosed   WCSessionStatus = "closed"
)

// WCResponse is a stub WalletConnect JSON-RPC response.
type WCResponse struct {
	ID     string
	Result []byte
	Error  string
}

// DIDDocument is a minimal DID document representation.
type DIDDocument struct {
	ID                 string
	Controller         []string
	VerificationMethod []DIDVerificationMethod
	Authentication     []string
	Service            []DIDService
	AlsoKnownAs        []string
}

// DIDVerificationMethod is a verification method entry.
type DIDVerificationMethod struct {
	ID           string
	Type         string
	Controller   string
	PublicKeyHex string
}

// DIDService is a service endpoint entry.
type DIDService struct {
	ID              string
	Type            string
	ServiceEndpoint string
}

// DIDResolver resolves DID strings to DID documents.
// Implementations include identity memory resolvers (ethr / web).
type DIDResolver interface {
	// Resolve fetches or constructs a DID document for the given DID URI.
	Resolve(ctx context.Context, did string) (*DIDDocument, error)

	// Method returns the DID method this resolver handles (e.g. "ethr", "web").
	Method() string
}
