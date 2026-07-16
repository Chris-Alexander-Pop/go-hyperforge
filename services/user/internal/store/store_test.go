package store_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/user/internal/store"
)

func TestUpsertAndGet(t *testing.T) {
	s := store.New()
	ctx := context.Background()

	p, err := s.Upsert(ctx, store.Profile{ID: "u1", Email: "u@example.com", Name: "U"})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != p.ID || got.Email != p.Email {
		t.Fatalf("mismatch: %+v vs %+v", got, p)
	}
	if _, err := s.Get(ctx, "missing"); err == nil {
		t.Fatal("expected not found")
	}
}
