package provider

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/iam"
)

// IdentityProvider defines the interface for authentication and token issuance.
// It acts as the central OIDC/SAML authority.
type IdentityProvider interface {
	// Authenticate validates credentials and returns a user.
	Authenticate(ctx context.Context, creds iam.Credentials) (*iam.User, error)

	// IssueToken generates a token for a given user.
	IssueToken(ctx context.Context, user *iam.User, scopes []string) (*iam.Token, error)

	// ValidateToken checks if a token is valid and returns the associated user claim.
	ValidateToken(ctx context.Context, token string) (*iam.User, error)

	// RevokeToken invalidates a token.
	RevokeToken(ctx context.Context, token string) error

	// CreateUser registers a new user.
	CreateUser(ctx context.Context, user iam.User, password string) (string, error)
}

// Config holds configuration for the Identity Provider.
type Config struct {
	// Driver specifies the IDP backend: "memory", "dex", "keycloak".
	Driver string `env:"IAM_DRIVER" env-default:"memory"`

	// Issuer is the URL of the issuer.
	Issuer string `env:"IAM_ISSUER" env-default:"https://idp.example.com"`
}
