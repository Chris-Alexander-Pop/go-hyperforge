/*
Package web3 defines Ethereum, IPFS, and SIWE interfaces for Web3 integrations.

# Scope (honest)

This root package provides:

  - Client — EVM JSON-RPC style client interface (balance, transfer, receipts)
  - Store — IPFS add/get/pin interface
  - Verifier — Sign-In with Ethereum (SIWE) verification interface
  - Shared message/receipt types, errors, and instrumented wrappers
  - In-memory adapters under adapters/memory for tests and local use

Concrete implementations:

  - adapters/geth — go-ethereum ethclient behind web3.Client
  - adapters/kubo — Kubo/IPFS HTTP API behind web3.Store
  - identity — SIWE cryptographic verification and basic DID parsing
  - blockchain/ethereum — thin wrapper re-exporting adapters/geth
  - blockchain/solana — JSON-RPC Solana client scaffold (not behind a root interface yet)
  - storage/ipfs — thin wrapper re-exporting adapters/kubo

WalletConnect session protocol is not implemented. DID support is limited to
string parse/format helpers (no resolver, no DID document fetch).
*/
package web3
