package password_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/auth/password"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/crypto"
)

func TestStoreSetAndAuthenticate(t *testing.T) {
	store := password.New(crypto.DefaultHashConfig())
	ctx := context.Background()

	if err := store.Set(ctx, "alice", "s3cret!"); err != nil {
		t.Fatal(err)
	}
	if !store.Hashed("alice") {
		t.Fatal("expected hashed password stored")
	}

	sub, err := store.Authenticate(ctx, "alice", "s3cret!")
	if err != nil {
		t.Fatal(err)
	}
	if sub != "alice" {
		t.Fatalf("subject=%s", sub)
	}

	if _, err := store.Authenticate(ctx, "alice", "wrong"); err == nil {
		t.Fatal("expected auth failure")
	}
	if _, err := store.Authenticate(ctx, "missing", "s3cret!"); err == nil {
		t.Fatal("expected missing user failure")
	}
}
