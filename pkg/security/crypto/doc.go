/*
Package crypto provides cryptographic primitives and helpers.

Features:
  - Encryption: AES-GCM (Encryptor) + envelope encryption via KeyProvider
  - Hashing: Argon2id / bcrypt password helpers
  - InstrumentedEncryptor for logging/tracing without leaking plaintext
  - PQC: hybrid KEM (X25519 + CIRCL ML-KEM / FIPS 203). Dilithium/ML-DSA
    signatures are not implemented — docs that claim them are outdated.

KeyProvider memory adapter: crypto/adapters/memory.
Cloud KMS backends are not shipped; use pkg/security/crypto/kms memory for
local encrypt/decrypt of small payloads.
*/
package crypto
