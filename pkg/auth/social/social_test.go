package social_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/social"
)

func TestNewAppleProvider(t *testing.T) {
	p, err := social.New(social.ProviderApple, "client-id", "jwt-client-secret", "https://example.com/callback")
	if err != nil {
		t.Fatal(err)
	}

	url := p.GetLoginURL("state-123")
	if !strings.Contains(url, "appleid.apple.com") {
		t.Fatalf("expected Apple auth URL, got %s", url)
	}
	if !strings.Contains(url, "response_mode=form_post") {
		t.Fatalf("expected form_post response_mode, got %s", url)
	}
}

func TestUserInfoFromIDToken(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"apple-user-1","email":"a@privaterelay.appleid.com"}`))
	idToken := "header." + payload + ".sig"

	info, err := social.UserInfoFromIDToken(idToken)
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != "apple-user-1" {
		t.Fatalf("unexpected id: %s", info.ID)
	}
	if info.Email != "a@privaterelay.appleid.com" {
		t.Fatalf("unexpected email: %s", info.Email)
	}
}

func TestUnsupportedProvider(t *testing.T) {
	_, err := social.New(social.ProviderType("linkedin"), "a", "b", "https://x")
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}
