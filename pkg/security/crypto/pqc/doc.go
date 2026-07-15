/*
Package pqc provides post-quantum cryptography helpers backed by Cloudflare CIRCL.

Status:
  - Hybrid KEM combines X25519 with ML-KEM (FIPS 203) via
    github.com/cloudflare/circl/kem/mlkem (levels 512/768/1024).
  - Dilithium / ML-DSA digital signatures via
    github.com/cloudflare/circl/sign/mldsa (ML-DSA-44/65/87) through
    Signer / Verifier (DilithiumSigner).

Suitable for hybrid key exchange and ML-DSA signing through KyberKEM /
HybridKEM and DilithiumSigner APIs.
*/
package pqc
