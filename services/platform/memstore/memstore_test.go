package memstore_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/platform/memstore"
)

func TestCRUD(t *testing.T) {
	s := memstore.New()
	ctx := context.Background()
	rec, err := s.Create(ctx, map[string]interface{}{"name": "demo"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Data["name"] != "demo" {
		t.Fatalf("unexpected data: %+v", got.Data)
	}
	if _, err := s.Update(ctx, rec.ID, map[string]interface{}{"name": "updated"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	list, err := s.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List: %v len=%d", err, len(list))
	}
	if err := s.Delete(ctx, rec.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
