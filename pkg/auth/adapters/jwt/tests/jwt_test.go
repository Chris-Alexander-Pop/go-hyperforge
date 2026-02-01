package jwt_test

import (
	"context"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/jwt"
	"testing"
	"time"
)

func TestJWTAdapter(t *testing.T) {
	cfg := jwt.Config{
		Secret:     "super-secret-key-for-testing",
		Expiration: time.Hour,
		Issuer:     "test-issuer",
	}

	adapter := jwt.New(cfg)
	userID := "user-123"
	roles := []string{"admin", "editor"}

	// 1. Generate Token
	token, err := adapter.Generate(userID, roles)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if token == "" {
		t.Fatal("Generated token is empty")
	}

	// 2. Verify Token
	ctx := context.Background()
	claims, err := adapter.Verify(ctx, token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// 3. Check Claims
	if claims.Subject != userID {
		t.Errorf("Expected subject %s, got %s", userID, claims.Subject)
	}
	if len(claims.Roles) != len(roles) {
		t.Errorf("Expected %d roles, got %d", len(roles), len(claims.Roles))
	} else {
		for i, r := range roles {
			if claims.Roles[i] != r {
				t.Errorf("Expected role %s at index %d, got %s", r, i, claims.Roles[i])
			}
		}
	}
	if claims.Issuer != cfg.Issuer {
		t.Errorf("Expected issuer %s, got %s", cfg.Issuer, claims.Issuer)
	}
}

func TestVerifyInvalidToken(t *testing.T) {
	cfg := jwt.Config{Secret: "secret"}
	adapter := jwt.New(cfg)

	_, err := adapter.Verify(context.Background(), "invalid-token-string")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}
