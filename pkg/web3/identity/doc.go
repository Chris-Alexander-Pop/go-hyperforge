/*
Package identity provides Sign-In with Ethereum (SIWE) verification and DID helpers.

# Scope (honest)

Implemented:

  - SIWE message formatting, nonce generation, and cryptographic signature recovery
    (via go-ethereum crypto). SIWEVerifier implements pkg/web3.Verifier and uses
    pkg/concurrency.SmartRWMutex for race-safe nonce tracking.
  - Basic DID string parse/format (did:method:identifier) and ethr DID construction
  - DID resolvers under adapters/memory (did:ethr synthesizer, did:web map, Registry)

Not implemented:

  - WalletConnect relay protocol (session stub lives under web3/adapters/memory)
  - Universal resolver / remote DID document HTTP fetch for did:web
  - ENS or other naming system lookups

Prefer pkg/web3.Verifier + adapters/memory for tests; use this package when you
need real EIP-191 signature recovery.
*/
package identity
