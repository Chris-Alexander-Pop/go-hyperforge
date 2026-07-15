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

  - identity — SIWE cryptographic verification (go-ethereum crypto) and basic DID parsing
  - blockchain/ethereum — geth ethclient wrapper (SDK-coupled; not yet behind Client)
  - blockchain/solana — JSON-RPC Solana client scaffold (not behind a root interface yet)
  - storage/ipfs — IPFS HTTP API client scaffold (not yet behind Store)

WalletConnect session protocol is not implemented. DID support is limited to
string parse/format helpers (no resolver, no DID document fetch). Prefer root
interfaces + memory adapters for new code; treat concrete SDK packages as
scaffolds until adapted.
*/
package web3
