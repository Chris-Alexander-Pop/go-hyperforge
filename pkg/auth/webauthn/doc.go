// Package webauthn provides WebAuthn (Passkeys) authentication support.
//
// It defines the interfaces for registration and login ceremonies compatible
// with the FIDO2/WebAuthn standard.
//
// Adapters:
//   - adapters/library — production path using github.com/go-webauthn/webauthn
//   - adapters/memory — in-memory test double with challenge tracking (not FIDO2-compliant)
package webauthn
