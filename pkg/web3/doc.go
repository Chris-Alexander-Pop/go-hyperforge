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
  - adapters/solana — Solana JSON-RPC behind web3.SolanaClient
  - adapters/kubo — Kubo/IPFS HTTP API behind web3.Store
  - adapters/memory — Ethereum/IPFS/SIWE/Solana/WalletConnect test doubles
  - identity — SIWE verification, DID parse/format, memory ethr/web DID resolvers
  - blockchain/ethereum — thin wrapper re-exporting adapters/geth
  - blockchain/solana — thin wrapper re-exporting adapters/solana
  - storage/ipfs — thin wrapper re-exporting adapters/kubo

WalletConnect is a session stub (no relay). DID resolution is in-memory ethr/web.
*/
package web3
