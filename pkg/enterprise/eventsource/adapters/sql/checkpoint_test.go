package sql_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
	eventsql "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource/adapters/sql"
	_ "modernc.org/sqlite"
)

func TestSQLCheckpointStore(t *testing.T) {
	db, err := sql.Open("sqlite", "file:proj_cp_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store, err := eventsql.New(db, eventsql.Config{Dialect: eventsql.DialectSQLite})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	cp, err := store.Load(ctx, "orders")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if cp.Position != 0 || cp.Name != "orders" {
		t.Fatalf("unexpected empty checkpoint %+v", cp)
	}

	if err := store.Save(ctx, eventsource.Checkpoint{
		Name:      "orders",
		Position:  42,
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cp, err = store.Load(ctx, "orders")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cp.Position != 42 {
		t.Fatalf("position want 42 got %d", cp.Position)
	}

	if err := store.Save(ctx, eventsource.Checkpoint{Name: "orders", Position: 100}); err != nil {
		t.Fatalf("Save upsert: %v", err)
	}
	cp, _ = store.Load(ctx, "orders")
	if cp.Position != 100 {
		t.Fatalf("upsert position want 100 got %d", cp.Position)
	}
}
