// Package memory provides in-memory Ethereum, IPFS, and SIWE adapters for testing.
//
// These adapters implement pkg/web3.Client, Store, and Verifier with no external
// RPC nodes or IPFS daemons. They use pkg/concurrency.SmartRWMutex for locking.
package memory
