package entraid_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/entraid"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestNewRequiresConfig(t *testing.T) {
	_, err := entraid.New(entraid.Config{})
	if !auth.IsInvalidConfig(err) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestVerifyEmptyToken(t *testing.T) {
	a, err := entraid.New(entraid.Config{
		TenantID: "common",
		ClientID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = a.Verify(context.Background(), "")
	if !errors.Is(err, auth.ErrInvalidToken) && !auth.IsInvalidToken(err) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestLoginEmptyCredentials(t *testing.T) {
	a, err := entraid.New(entraid.Config{
		TenantID: "common",
		ClientID: "00000000-0000-0000-0000-000000000000",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = a.Login(context.Background(), "user", "")
	if !auth.IsInvalidCredentials(err) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}
