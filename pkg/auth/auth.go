// Package auth provides authentication and authorization primitives.
//
// Supported adapters:
//   - Local: Username/password with bcrypt
//   - OIDC: OpenID Connect integration
//   - Session: Server-side session management
//   - PASETO: Secure token generation
//
// Features:
//   - Unified Claims structure for identity
//   - Token verification interface
//   - MFA support (TOTP, WebAuthn)
//   - Social login adapters
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/oidc"
//
//	verifier := oidc.New(cfg)
//	claims, err := verifier.Verify(ctx, token)
package auth

import (
	"context"
)

// Claims represents the standard identity claims
type Claims struct {
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`

	// Extended
	Email    string                 `json:"email,omitempty"`
	Role     string                 `json:"role,omitempty"` // Standardize on "role" or "groups"
	Metadata map[string]interface{} `json:"-"`              // Catch-all?
}

// Verifier is responsible for validating an access token / ID token
type Verifier interface {
	Verify(ctx context.Context, token string) (*Claims, error)
}
