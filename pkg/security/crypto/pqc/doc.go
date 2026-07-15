/*
Package pqc provides post-quantum cryptography helpers backed by Cloudflare CIRCL.

Status:
  - Hybrid KEM combines X25519 with ML-KEM (FIPS 203) via
    github.com/cloudflare/circl/kem/mlkem (levels 512/768/1024).
  - Dilithium / ML-DSA digital signatures are NOT implemented.

Suitable for hybrid key exchange prototyping and production ML-KEM use through
the KyberKEM / HybridKEM APIs. Signature algorithms remain out of scope.
*/
package pqc
