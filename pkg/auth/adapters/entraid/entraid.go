package entraid

import (
	"context"
	"strings"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
	pkgauth "github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
)

// Config holds configuration for Azure EntraID (formerly AD).
type Config struct {
	TenantID string `env:"AUTH_ENTRAID_TENANT_ID" validate:"required"`
	ClientID string `env:"AUTH_ENTRAID_CLIENT_ID" validate:"required"`
	// Authority URL (optional, defaults to standard Azure cloud)
	Authority string `env:"AUTH_ENTRAID_AUTHORITY"`
	// Issuer overrides OIDC issuer discovery (defaults to {Authority}/v2.0).
	Issuer string `env:"AUTH_ENTRAID_ISSUER"`
}

// Adapter implements auth.IdentityProvider and auth.Verifier for EntraID.
type Adapter struct {
	client   public.Client
	clientID string
	issuer   string

	verifierOnce sync.Once
	verifier     *oidc.IDTokenVerifier
	verifierErr  error
}

// New creates a new EntraID adapter.
func New(cfg Config) (*Adapter, error) {
	if cfg.TenantID == "" || cfg.ClientID == "" {
		return nil, pkgauth.ErrInvalidConfigMsg("TenantID and ClientID are required", nil)
	}

	authority := cfg.Authority
	if authority == "" {
		authority = "https://login.microsoftonline.com/" + cfg.TenantID
	}
	authority = strings.TrimRight(authority, "/")

	issuer := cfg.Issuer
	if issuer == "" {
		issuer = authority + "/v2.0"
	}

	client, err := public.New(cfg.ClientID, public.WithAuthority(authority))
	if err != nil {
		return nil, errors.Internal("failed to create msal client", err)
	}

	return &Adapter{
		client:   client,
		clientID: cfg.ClientID,
		issuer:   issuer,
	}, nil
}

// Login authenticates a user with username and password (ROPC).
func (a *Adapter) Login(ctx context.Context, username, password string) (*pkgauth.Claims, error) {
	if username == "" || password == "" {
		return nil, pkgauth.ErrInvalidCredentials
	}

	scopes := []string{"User.Read"}

	//lint:ignore SA1019 ROPC flow required for IdentityProvider.Login
	result, err := a.client.AcquireTokenByUsernamePassword(ctx, scopes, username, password) //nolint:staticcheck
	if err != nil {
		return nil, errors.Unauthorized("entra id login failed", err)
	}

	rawID := ""
	if result.IDToken.RawToken != "" {
		rawID = result.IDToken.RawToken
	}
	if rawID != "" {
		claims, verr := a.Verify(ctx, rawID)
		if verr == nil {
			if claims.Metadata == nil {
				claims.Metadata = map[string]interface{}{}
			}
			claims.Metadata["access_token"] = result.AccessToken
			return claims, nil
		}
	}

	return &pkgauth.Claims{
		Subject:   result.Account.HomeAccountID,
		Issuer:    a.issuer,
		ExpiresAt: result.ExpiresOn.Unix(),
		Email:     result.Account.PreferredUsername,
		Metadata: map[string]interface{}{
			"access_token": result.AccessToken,
		},
	}, nil
}

// Verify validates an EntraID ID token via OIDC discovery + JWKS.
func (a *Adapter) Verify(ctx context.Context, token string) (*pkgauth.Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, pkgauth.ErrInvalidToken
	}

	verifier, err := a.getVerifier(ctx)
	if err != nil {
		return nil, errors.Internal("failed to initialize entra oidc verifier", err)
	}

	idToken, err := verifier.Verify(ctx, token)
	if err != nil {
		return nil, pkgauth.ErrInvalidTokenWrap(err)
	}

	var raw struct {
		Email             string   `json:"email"`
		PreferredUsername string   `json:"preferred_username"`
		Name              string   `json:"name"`
		Roles             []string `json:"roles"`
		Groups            []string `json:"groups"`
		OID               string   `json:"oid"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return nil, errors.Internal("failed to parse entra claims", err)
	}

	email := raw.Email
	if email == "" {
		email = raw.PreferredUsername
	}
	roles := append([]string{}, raw.Roles...)
	roles = append(roles, raw.Groups...)

	return &pkgauth.Claims{
		Subject:   idToken.Subject,
		Issuer:    idToken.Issuer,
		Audience:  idToken.Audience,
		ExpiresAt: idToken.Expiry.Unix(),
		IssuedAt:  idToken.IssuedAt.Unix(),
		Email:     email,
		Roles:     roles,
		Metadata: map[string]interface{}{
			"oid":  raw.OID,
			"name": raw.Name,
		},
	}, nil
}

func (a *Adapter) getVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	a.verifierOnce.Do(func() {
		provider, err := oidc.NewProvider(ctx, a.issuer)
		if err != nil {
			a.verifierErr = err
			return
		}
		a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})
	})
	return a.verifier, a.verifierErr
}

var (
	_ pkgauth.IdentityProvider = (*Adapter)(nil)
	_ pkgauth.Verifier         = (*Adapter)(nil)
)
