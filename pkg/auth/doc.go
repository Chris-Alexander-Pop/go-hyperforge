// Package auth provides core authentication interfaces and types.
//
// This package defines the common contracts for:
//   - Identity Providers (IdP)
//   - Token validation (JWT, Paseto, OIDC)
//   - Session management (encrypt metadata with EncryptionKey)
//   - User context propagation
//   - MFA (TOTP + SMS/email channel via pkg/communication)
//   - Local passwords via pkg/auth/password (crypto.Hasher / Argon2id)
//   - Social OAuth2 (Google, GitHub, Facebook, Apple)
//   - WebAuthn (library adapter for production; memory for tests)
//   - SAML SP client skeleton (pkg/auth/saml; memory ACS test double)
//
// OAuth2 authorization-server shapes (TokenIssuer, Authorize/Token) live in
// package oauth2 with an in-memory adapter — enough for local token generation,
// not a full OpenID Provider. Memory OAuth2 client secrets are hashed with
// crypto.Hasher at rest.
//
// It serves as the foundation for specific adapters located in subpackages.
package auth
