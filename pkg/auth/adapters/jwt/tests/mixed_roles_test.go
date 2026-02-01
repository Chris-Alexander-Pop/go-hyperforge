package jwt_test

import (
	"context"
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/jwt"
	libjwt "github.com/golang-jwt/jwt/v5"
)

func TestVerifyMixedRoles(t *testing.T) {
	cfg := jwt.Config{
		Secret:     "super-secret-key-for-testing",
		Expiration: time.Hour,
		Issuer:     "test-issuer",
	}

	adapter := jwt.New(cfg)

	// Manually create a token with both "role" and "roles"
	claims := libjwt.MapClaims{
		"sub":   "user-mixed",
		"iss":   cfg.Issuer,
		"role":  "single-role",
		"roles": []interface{}{"array-role-1", "array-role-2"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	tokenObj := libjwt.NewWithClaims(libjwt.SigningMethodHS256, claims)
	tokenString, err := tokenObj.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Verify
	ctx := context.Background()
	authClaims, err := adapter.Verify(ctx, tokenString)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	// Check if all roles are present
	expectedRoles := []string{"single-role", "array-role-1", "array-role-2"}

	found := make(map[string]bool)
	for _, r := range authClaims.Roles {
		found[r] = true
	}

	for _, r := range expectedRoles {
		if !found[r] {
			t.Errorf("Expected role %s not found in claims", r)
		}
	}
}
