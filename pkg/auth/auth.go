// Package auth provides authentication and authorization primitives.
//
// Supported adapters:
//   - JWT: Local JWT generation and verification
//   - OIDC: OpenID Connect integration
//   - Session: Server-side session management
//   - PASETO: Secure token generation
package auth

import (
	"context"
)

// Config configures the auth package.
type Config struct {
	// Adapter specifies which auth adapter to use (jwt, oidc, session).
	Adapter string `env:"AUTH_ADAPTER" env-default:"jwt"`

	// JWT Config (if Adapter == jwt)
	// Note: In a real app, this might be nested or handled by the specific adapter's New function.
	// For simplicity with the standard pattern, we might define generic config here or let New() take a specific config.
	// The standard says "Config struct with env tags".
}

// Claims represents the standard identity claims
type Claims struct {
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`

	// Extended
	Email    string                 `json:"email,omitempty"`
	Roles    []string               `json:"roles,omitempty"` // Standardize on "role" or "groups"
	Metadata map[string]interface{} `json:"-"`               // Catch-all
}

// IdentityProvider defines the interface for authentication providers.
type IdentityProvider interface {
	// Login authenticates a user with credentials.
	Login(ctx context.Context, username, password string) (*Claims, error)
}

// Verifier is responsible for validating an access token / ID token.
type Verifier interface {
	Verify(ctx context.Context, token string) (*Claims, error)
}
