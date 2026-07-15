package oidc_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/oidc"
)

func TestNewRequiresConfig(t *testing.T) {
	_, err := oidc.New(context.Background(), oidc.Config{})
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestExchangeRequiresClientSecret(t *testing.T) {
	// Discovery hits the network; skip if issuer unreachable.
	a, err := oidc.New(context.Background(), oidc.Config{
		IssuerURL: "https://accounts.google.com",
		ClientID:  "test-client",
	})
	if err != nil {
		t.Skipf("oidc discovery unavailable: %v", err)
	}
	_, err = a.Exchange(context.Background(), "code")
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config for missing secret/redirect, got %v", err)
	}
	_, err = a.AuthCodeURL("state")
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config for AuthCodeURL, got %v", err)
	}
}

func TestVerifyEmptyToken(t *testing.T) {
	a, err := oidc.New(context.Background(), oidc.Config{
		IssuerURL: "https://accounts.google.com",
		ClientID:  "test-client",
	})
	if err != nil {
		t.Skipf("oidc discovery unavailable: %v", err)
	}
	_, err = a.Verify(context.Background(), "")
	if !auth.IsInvalidToken(err) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
