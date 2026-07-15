package memory_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/auth"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/oidc"
	"github.com/chris-alexander-pop/system-design-library/pkg/auth/adapters/oidc/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestMemoryExchanger(t *testing.T) {
	ctx := context.Background()
	ex := memory.New("client")
	ex.Seed("abc", &oidc.TokenSet{
		AccessToken: "access",
		IDToken:     "id",
		Claims:      &auth.Claims{Subject: "u1", Email: "u1@example.com"},
	})

	url, err := ex.AuthCodeURL("state-1")
	if err != nil || url == "" {
		t.Fatalf("AuthCodeURL: %v %q", err, url)
	}

	ts, err := ex.Exchange(ctx, "abc")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if ts.AccessToken != "access" || ts.Claims.Subject != "u1" {
		t.Fatalf("unexpected token set: %+v", ts)
	}

	_, err = ex.Exchange(ctx, "abc")
	if !errors.Is(err, auth.ErrExchangeFailed) {
		t.Fatalf("expected reuse to fail, got %v", err)
	}
}
