package memory_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/oauth2"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/oauth2/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestAuthorizationCodeFlow(t *testing.T) {
	ctx := context.Background()
	srv := memory.New(oauth2.Config{
		Issuer:          "test-issuer",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
		AuthCodeTTL:     5 * time.Minute,
	})

	err := srv.RegisterClient(ctx, oauth2.Client{
		ID:           "app",
		Secret:       "secret",
		RedirectURIs: []string{"https://app.example/callback"},
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantAuthorizationCode,
			oauth2.GrantRefreshToken,
		},
		Scopes: []string{"read", "write"},
	})
	if err != nil {
		t.Fatalf("RegisterClient: %v", err)
	}

	authz, err := srv.Authorize(ctx, oauth2.AuthorizeRequest{
		ResponseType: oauth2.ResponseTypeCode,
		ClientID:     "app",
		RedirectURI:  "https://app.example/callback",
		Scope:        []string{"read"},
		State:        "xyz",
		Subject:      "user-1",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if authz.Code == "" || authz.State != "xyz" {
		t.Fatalf("unexpected authorize response: %+v", authz)
	}

	tok, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantAuthorizationCode,
		Code:         authz.Code,
		RedirectURI:  "https://app.example/callback",
		ClientID:     "app",
		ClientSecret: "secret",
	})
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if tok.AccessToken == "" || tok.TokenType != "Bearer" || tok.RefreshToken == "" {
		t.Fatalf("unexpected token response: %+v", tok)
	}

	claims, err := srv.Introspect(ctx, tok.AccessToken)
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	if claims.Subject != "user-1" || claims.ClientID != "app" || claims.Issuer != "test-issuer" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Code reuse must fail
	_, err = srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantAuthorizationCode,
		Code:         authz.Code,
		RedirectURI:  "https://app.example/callback",
		ClientID:     "app",
		ClientSecret: "secret",
	})
	if err == nil {
		t.Fatal("expected reused code to fail")
	}
	if !errors.Is(err, oauth2.ErrInvalidGrant) {
		t.Fatalf("expected ErrInvalidGrant, got %v", err)
	}
}

func TestClientCredentialsAndRefresh(t *testing.T) {
	ctx := context.Background()
	srv := memory.New(oauth2.Config{AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour})
	_ = srv.RegisterClient(ctx, oauth2.Client{
		ID:     "svc",
		Secret: "s3cret",
		GrantTypes: []oauth2.GrantType{
			oauth2.GrantClientCredentials,
			oauth2.GrantRefreshToken,
			oauth2.GrantAuthorizationCode,
		},
		RedirectURIs: []string{"https://cb"},
		Scopes:       []string{"api"},
	})

	tok, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantClientCredentials,
		ClientID:     "svc",
		ClientSecret: "s3cret",
		Scope:        []string{"api"},
	})
	if err != nil {
		t.Fatalf("client_credentials: %v", err)
	}
	if tok.RefreshToken != "" {
		t.Fatal("client_credentials should not issue refresh tokens")
	}

	authz, err := srv.Authorize(ctx, oauth2.AuthorizeRequest{
		ClientID:    "svc",
		RedirectURI: "https://cb",
		Subject:     "u",
		Scope:       []string{"api"},
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	tok2, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantAuthorizationCode,
		Code:         authz.Code,
		RedirectURI:  "https://cb",
		ClientID:     "svc",
		ClientSecret: "s3cret",
	})
	if err != nil {
		t.Fatalf("auth code: %v", err)
	}

	refreshed, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantRefreshToken,
		RefreshToken: tok2.RefreshToken,
		ClientID:     "svc",
		ClientSecret: "s3cret",
	})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.AccessToken == tok2.AccessToken {
		t.Fatalf("expected new access token, got %+v", refreshed)
	}

	// Old refresh revoked
	_, err = srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantRefreshToken,
		RefreshToken: tok2.RefreshToken,
		ClientID:     "svc",
		ClientSecret: "s3cret",
	})
	if !errors.Is(err, oauth2.ErrInvalidGrant) {
		t.Fatalf("expected revoked refresh to fail, got %v", err)
	}
}

func TestPKCEAndRevoke(t *testing.T) {
	ctx := context.Background()
	srv := memory.New(oauth2.Config{})
	_ = srv.RegisterClient(ctx, oauth2.Client{
		ID:           "pkce",
		Secret:       "",
		RedirectURIs: []string{"https://native/cb"},
		GrantTypes:   []oauth2.GrantType{oauth2.GrantAuthorizationCode},
	})

	verifier := "challenge-verifier-value-1234567890"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	authz, err := srv.Authorize(ctx, oauth2.AuthorizeRequest{
		ClientID:            "pkce",
		RedirectURI:         "https://native/cb",
		Subject:             "user",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
	})
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	_, err = srv.Token(ctx, oauth2.TokenRequest{
		GrantType:   oauth2.GrantAuthorizationCode,
		Code:        authz.Code,
		RedirectURI: "https://native/cb",
		ClientID:    "pkce",
	})
	if !errors.Is(err, oauth2.ErrInvalidGrant) {
		t.Fatalf("expected PKCE failure without verifier, got %v", err)
	}

	// Re-authorize since code was not marked used on PKCE failure... actually looking at code,
	// PKCE check happens after Used check but before Used=true. Wait - if PKCE fails, Used is not set.
	// But we already looked up the same code - good, code still valid.

	tok, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantAuthorizationCode,
		Code:         authz.Code,
		RedirectURI:  "https://native/cb",
		ClientID:     "pkce",
		CodeVerifier: verifier,
	})
	if err != nil {
		t.Fatalf("Token with PKCE: %v", err)
	}

	if err := srv.Revoke(ctx, tok.AccessToken); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	_, err = srv.Introspect(ctx, tok.AccessToken)
	if err == nil {
		t.Fatal("expected introspect to fail after revoke")
	}
}

func TestInvalidClientAndPasswordGrant(t *testing.T) {
	ctx := context.Background()
	authn := passwordAuth(func(ctx context.Context, u, p string) (string, error) {
		if u == "alice" && p == "pw" {
			return "alice-id", nil
		}
		return "", errors.Unauthorized("bad", nil)
	})
	srv := memory.New(oauth2.Config{}, memory.WithPasswordAuthenticator(authn))
	_ = srv.RegisterClient(ctx, oauth2.Client{
		ID:         "c",
		Secret:     "s",
		GrantTypes: []oauth2.GrantType{oauth2.GrantPassword},
	})

	_, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantPassword,
		ClientID:     "c",
		ClientSecret: "wrong",
		Username:     "alice",
		Password:     "pw",
	})
	if !errors.Is(err, oauth2.ErrInvalidClient) {
		t.Fatalf("expected invalid client, got %v", err)
	}

	tok, err := srv.Token(ctx, oauth2.TokenRequest{
		GrantType:    oauth2.GrantPassword,
		ClientID:     "c",
		ClientSecret: "s",
		Username:     "alice",
		Password:     "pw",
	})
	if err != nil {
		t.Fatalf("password grant: %v", err)
	}
	claims, err := srv.Introspect(ctx, tok.AccessToken)
	if err != nil || claims.Subject != "alice-id" {
		t.Fatalf("claims: %+v err=%v", claims, err)
	}
}

func TestIssueAccessTokenDirect(t *testing.T) {
	ctx := context.Background()
	srv := memory.New(oauth2.Config{Issuer: "direct"})
	tok, err := srv.IssueAccessToken(ctx, "sub", "cli", []string{"a"})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	claims, err := srv.Issuer().Introspect(ctx, tok.AccessToken)
	if err != nil {
		t.Fatalf("Introspect: %v", err)
	}
	if claims.Subject != "sub" {
		t.Fatalf("subject=%s", claims.Subject)
	}
}

type passwordAuth func(ctx context.Context, username, password string) (string, error)

func (f passwordAuth) Authenticate(ctx context.Context, username, password string) (string, error) {
	return f(ctx, username, password)
}
