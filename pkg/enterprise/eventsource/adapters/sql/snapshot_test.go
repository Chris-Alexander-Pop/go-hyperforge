package sql_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
	eventsql "github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource/adapters/sql"
	_ "modernc.org/sqlite"
)

func TestSQLSnapshotStore(t *testing.T) {
	db, err := sql.Open("sqlite", "file:agg_snap_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	store, err := eventsql.NewSnapshotStore(db, eventsql.Config{Dialect: eventsql.DialectSQLite})
	if err != nil {
		t.Fatalf("NewSnapshotStore: %v", err)
	}
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	missing, err := store.Load(ctx, "missing")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil snapshot, got %+v", missing)
	}

	ts := time.Now().UTC().Truncate(time.Second)
	if err := store.Save(ctx, eventsource.Snapshot{
		AggregateID:   "order-1",
		AggregateType: "Order",
		Version:       5,
		Timestamp:     ts,
		Data:          json.RawMessage(`{"total":42,"status":"open"}`),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, "order-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected snapshot")
	}
	if loaded.AggregateID != "order-1" || loaded.AggregateType != "Order" {
		t.Fatalf("identity: %+v", loaded)
	}
	if loaded.Version != 5 {
		t.Fatalf("version want 5 got %d", loaded.Version)
	}
	if string(loaded.Data) != `{"total":42,"status":"open"}` {
		t.Fatalf("data=%s", loaded.Data)
	}

	// Upsert keeps latest only.
	if err := store.Save(ctx, eventsource.Snapshot{
		AggregateID:   "order-1",
		AggregateType: "Order",
		Version:       10,
		Data:          json.RawMessage(`{"total":99,"status":"paid"}`),
	}); err != nil {
		t.Fatalf("Save upsert: %v", err)
	}
	loaded, err = store.Load(ctx, "order-1")
	if err != nil || loaded == nil {
		t.Fatalf("Load after upsert: %v %v", loaded, err)
	}
	if loaded.Version != 10 {
		t.Fatalf("upsert version want 10 got %d", loaded.Version)
	}
	if string(loaded.Data) != `{"total":99,"status":"paid"}` {
		t.Fatalf("upsert data=%s", loaded.Data)
	}
}

func TestSQLSnapshotStoreRequiresDB(t *testing.T) {
	_, err := eventsql.NewSnapshotStore(nil, eventsql.Config{})
	if err == nil {
		t.Fatal("expected error for nil db")
	}
}
