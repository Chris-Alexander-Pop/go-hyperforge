// Package auth provides core authentication interfaces and types.
//
// This package defines the common contracts for:
//   - Identity Providers (IdP)
//   - Token validation (JWT, Paseto, OIDC)
//   - Session management
//   - User context propagation
//
// OAuth2 authorization-server shapes (TokenIssuer, Authorize/Token) live in
// package oauth2 with an in-memory adapter — enough for local token generation,
// not a full OpenID Provider.
//
// It serves as the foundation for specific adapters located in subpackages.
package auth
