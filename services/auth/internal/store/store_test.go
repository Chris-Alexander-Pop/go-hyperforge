package store_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/auth/internal/store"
)

func TestRegisterAndAuthenticate(t *testing.T) {
	s := store.New()
	ctx := context.Background()

	acct, err := s.Register(ctx, "a@example.com", "secret123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if acct.UserID == "" || acct.Email != "a@example.com" {
		t.Fatalf("unexpected account: %+v", acct)
	}

	if _, err := s.Register(ctx, "a@example.com", "other"); err == nil {
		t.Fatal("expected conflict on duplicate email")
	}

	got, err := s.Authenticate(ctx, "a@example.com", "secret123")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if got.UserID != acct.UserID {
		t.Fatalf("user id mismatch: %s vs %s", got.UserID, acct.UserID)
	}

	if _, err := s.Authenticate(ctx, "a@example.com", "wrong"); err == nil {
		t.Fatal("expected auth failure")
	}
}
