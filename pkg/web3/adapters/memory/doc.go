// Package memory provides in-memory Ethereum, IPFS, SIWE, Solana, and
// WalletConnect adapters for testing.
//
// These adapters implement pkg/web3.Client, Store, Verifier, SolanaClient, and
// WalletConnectSession with no external RPC nodes or relays. They use
// pkg/concurrency.SmartRWMutex for locking.
package memory
