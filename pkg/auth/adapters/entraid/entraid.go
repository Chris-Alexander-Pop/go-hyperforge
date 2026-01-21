package entraid

import (
	"context"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	pkgauth "github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds configuration for Azure EntraID (formerly AD).
type Config struct {
	TenantID string `env:"AUTH_ENTRAID_TENANT_ID" validate:"required"`
	ClientID string `env:"AUTH_ENTRAID_CLIENT_ID" validate:"required"`
	// Authority URL (optional, defaults to standard Azure cloud)
	Authority string `env:"AUTH_ENTRAID_AUTHORITY"`
}

// Adapter implements auth.IdentityProvider and auth.Verifier for EntraID.
type Adapter struct {
	client public.Client
}

// New creates a new EntraID adapter.
func New(cfg Config) (*Adapter, error) {
	authority := cfg.Authority
	if authority == "" {
		authority = "https://login.microsoftonline.com/" + cfg.TenantID
	}

	client, err := public.New(cfg.ClientID, public.WithAuthority(authority))
	if err != nil {
		return nil, errors.Internal("failed to create msal client", err)
	}

	return &Adapter{
		client: client,
	}, nil
}

// Login authenticates a user with username and password.
func (a *Adapter) Login(ctx context.Context, username, password string) (*pkgauth.Claims, error) {
	// Use ROPC flow (Resource Owner Password Credentials)
	scopes := []string{"User.Read"} // Default scope

	result, err := a.client.AcquireTokenByUsernamePassword(ctx, scopes, username, password) //lint:ignore SA1019 //nolint:staticcheck // ROPC flow required
	if err != nil {
		return nil, errors.Unauthorized("entra id login failed", err)
	}

	// Helper to extract claims from the result
	return &pkgauth.Claims{
		Subject:   result.Account.HomeAccountID,
		Issuer:    "entraid", // Can be refining this from result.IDToken if parsed
		ExpiresAt: result.ExpiresOn.Unix(),
		Metadata: map[string]interface{}{
			"access_token": result.AccessToken,
		},
	}, nil
}

// Verify validates an EntraID token.
func (a *Adapter) Verify(ctx context.Context, token string) (*pkgauth.Claims, error) {
	// MSAL Go is primarily for acquiring tokens, validating them usually requires a separate library or manual JWT validation with keys from JWKS.
	// Since we don't have a dedicated validator library set up specifically for this adapter in the imports list
	// (other than what's available globally like golang-jwt/jwt or go-oidc),
	// we will stub this or use a generic JWT validation if keys were provided.

	// Return stub for now to satisfy interface.
	return nil, errors.Unimplemented("entraid token verification not implemented", nil)
}
