package oidc

import (
	"context"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config configures the OIDC adapter.
//
// Verify-only mode needs IssuerURL + ClientID.
// Authorization-code exchange additionally needs ClientSecret + RedirectURL.
type Config struct {
	IssuerURL    string `env:"OIDC_ISSUER_URL"`
	ClientID     string `env:"OIDC_CLIENT_ID"`
	ClientSecret string `env:"OIDC_CLIENT_SECRET"`
	RedirectURL  string `env:"OIDC_REDIRECT_URL"`
	Scopes       []string
}

// TokenSet is the result of an authorization-code exchange.
type TokenSet struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
	Expiry       int64
	TokenType    string
	Claims       *auth.Claims
}

// CodeExchanger exchanges an authorization code for tokens and verified claims.
type CodeExchanger interface {
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*TokenSet, error)
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) (string, error)
}

// Adapter verifies OIDC ID tokens and optionally exchanges authorization codes.
type Adapter struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth2   *oauth2.Config
}

// New creates an OIDC verifier. Exchange/AuthCodeURL require ClientSecret + RedirectURL.
func New(ctx context.Context, cfg Config) (*Adapter, error) {
	if cfg.IssuerURL == "" || cfg.ClientID == "" {
		return nil, auth.ErrInvalidConfigMsg("IssuerURL and ClientID are required", nil)
	}

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize oidc provider")
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
	})

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	a := &Adapter{
		provider: provider,
		verifier: verifier,
	}

	if cfg.ClientSecret != "" && cfg.RedirectURL != "" {
		a.oauth2 = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       scopes,
		}
	}

	return a, nil
}

// Verify validates an ID token string.
func (a *Adapter) Verify(ctx context.Context, tokenString string) (*auth.Claims, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil, auth.ErrInvalidToken
	}

	idToken, err := a.verifier.Verify(ctx, tokenString)
	if err != nil {
		return nil, auth.ErrInvalidTokenWrap(err)
	}
	return mapClaims(idToken)
}

// AuthCodeURL returns the provider authorization URL.
// Returns ErrInvalidConfig when ClientSecret/RedirectURL were not configured.
func (a *Adapter) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) (string, error) {
	if a.oauth2 == nil {
		return "", auth.ErrInvalidConfigMsg("ClientSecret and RedirectURL required for authorization-code flow", nil)
	}
	return a.oauth2.AuthCodeURL(state, opts...), nil
}

// Exchange redeems an authorization code, verifies the ID token when present,
// and returns a TokenSet. Without ClientSecret/RedirectURL this returns ErrInvalidConfig.
func (a *Adapter) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*TokenSet, error) {
	if a.oauth2 == nil {
		return nil, auth.ErrInvalidConfigMsg("ClientSecret and RedirectURL required for code exchange", nil)
	}
	if strings.TrimSpace(code) == "" {
		return nil, auth.ErrExchangeFailed
	}

	tok, err := a.oauth2.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, errors.New(auth.CodeExchangeFailed, "oidc code exchange failed", err)
	}

	ts := &TokenSet{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       tok.Expiry.Unix(),
	}
	if raw, ok := tok.Extra("id_token").(string); ok && raw != "" {
		ts.IDToken = raw
		claims, verr := a.Verify(ctx, raw)
		if verr != nil {
			return nil, verr
		}
		ts.Claims = claims
	}
	return ts, nil
}

func mapClaims(idToken *oidc.IDToken) (*auth.Claims, error) {
	var claims struct {
		Email    string   `json:"email"`
		Verified bool     `json:"email_verified"`
		Role     string   `json:"role"`
		Roles    []string `json:"roles"`
		Groups   []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, errors.Wrap(err, "failed to parse oidc claims")
	}

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
		Roles:     allRoles,
	}, nil
}

var (
	_ auth.Verifier = (*Adapter)(nil)
	_ CodeExchanger = (*Adapter)(nil)
)
