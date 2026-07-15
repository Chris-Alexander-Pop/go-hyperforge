/*
Package pqc provides experimental post-quantum cryptography helpers.

Status (honest):
  - Hybrid KEM combines X25519 with a *demo* Kyber-like KEM (not a vetted
    production ML-KEM). Prefer Cloudflare CIRCL / liboqs for real deployments.
  - Dilithium / ML-DSA digital signatures are NOT implemented. Earlier docs
    that listed Dilithium as included were overclaims.

Use for learning and interface prototyping only.
*/
package pqc
