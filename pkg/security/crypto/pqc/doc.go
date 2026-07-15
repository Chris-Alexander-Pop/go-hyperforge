/*
Package pqc provides experimental post-quantum cryptography helpers.

EXPERIMENTAL — not for production confidentiality.

Status (honest):
  - Hybrid KEM combines X25519 with a *demo* Kyber-like KEM (not a vetted
    production ML-KEM). Cloudflare CIRCL / liboqs are NOT vendored in go.mod;
    integrate them in adapters when ready.
  - Dilithium / ML-DSA digital signatures are NOT implemented. Earlier docs
    that listed Dilithium as included were overclaims.

Use for learning and interface prototyping only.
*/
package pqc
