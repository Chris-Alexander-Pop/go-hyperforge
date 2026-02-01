package oidc

import (
	"context"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
)

type Config struct {
	IssuerURL string `env:"OIDC_ISSUER_URL"` // e.g. https://id.google.com or https://cognito...
	ClientID  string `env:"OIDC_CLIENT_ID"`  // The Audience to verify
}

type Adapter struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
}

func New(ctx context.Context, cfg Config) (*Adapter, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize oidc provider")
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	return &Adapter{
		provider: provider,
		verifier: verifier,
	}, nil
}

func (a *Adapter) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	idToken, err := a.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, errors.Wrap(err, "oidc token verification failed")
	}

	// Extract Claims
	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Role     string   `json:"role"`
		Roles    []string `json:"roles"`
		Groups   []string `json:"groups"` // Common alternative
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.Wrap(err, "failed to parse oidc claims")
	}

	// Consolidate roles
	var allRoles []string
	if claims.Role != "" {
		allRoles = append(allRoles, claims.Role)
	}
	allRoles = append(allRoles, claims.Roles...)
	allRoles = append(allRoles, claims.Groups...)

	return &auth.Claims{
		Subject:   idToken.Subject,
		Issuer:    idToken.Issuer,
		Audience:  idToken.Audience,
		ExpiresAt: idToken.Expiry.Unix(),
		IssuedAt:  idToken.IssuedAt.Unix(),
		Email:     claims.Email,
		Roles:     allRoles, // Mapping might depend on provider (e.g. "cognito:groups")
	}, nil
}
