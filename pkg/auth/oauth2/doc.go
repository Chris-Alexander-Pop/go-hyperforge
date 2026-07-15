// Package oauth2 provides authorization-server oriented interfaces for issuing
// OAuth 2.0 access tokens (authorization-code and client-credentials grants).
//
// This is enough for catalog "OAuth2 token generation" and local testing via
// adapters/memory. It is not a full OpenID Provider (no discovery document,
// userinfo, or ID token issuance beyond an optional opaque id_token field).
package oauth2
