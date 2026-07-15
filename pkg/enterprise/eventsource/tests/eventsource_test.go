package eventsource_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/enterprise/eventsource/adapters/memory"
	pkgerrors "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/events"
	eventsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/events/adapters/memory"
)

func TestAppendAndLoad(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	evts := []eventsource.Event{
		{EventType: "OrderCreated", AggregateType: "Order", Data: json.RawMessage(`{"id":"1"}`)},
		{EventType: "ItemAdded", AggregateType: "Order", Data: json.RawMessage(`{"sku":"A"}`)},
	}
	if err := store.Append(ctx, "order-1", 0, evts); err != nil {
		t.Fatalf("Append: %v", err)
	}

	loaded, err := store.Load(ctx, "order-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
	if loaded[0].Version != 1 || loaded[1].Version != 2 {
		t.Fatalf("expected 1-based versions 1,2 got %d,%d", loaded[0].Version, loaded[1].Version)
	}
	if loaded[0].AggregateID != "order-1" {
		t.Fatalf("expected AggregateID set, got %q", loaded[0].AggregateID)
	}
}

func TestAppendConcurrencyConflict(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	if err := store.Append(ctx, "agg", 0, []eventsource.Event{{EventType: "A"}}); err != nil {
		t.Fatalf("first Append: %v", err)
	}

	err := store.Append(ctx, "agg", 0, []eventsource.Event{{EventType: "B"}})
	if err == nil {
		t.Fatal("expected version conflict")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != eventsource.CodeVersionConflict {
		t.Fatalf("expected version conflict AppError, got %v", err)
	}

	// Correct expected version succeeds.
	if err := store.Append(ctx, "agg", 1, []eventsource.Event{{EventType: "B"}}); err != nil {
		t.Fatalf("Append with expected=1: %v", err)
	}
}

func TestAppendSkipConcurrencyCheck(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	if err := store.Append(ctx, "agg", 0, []eventsource.Event{{EventType: "A"}}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// expectedVersion < 0 skips check
	if err := store.Append(ctx, "agg", -1, []eventsource.Event{{EventType: "B"}}); err != nil {
		t.Fatalf("Append skip check: %v", err)
	}
	loaded, _ := store.Load(ctx, "agg")
	if len(loaded) != 2 {
		t.Fatalf("expected 2 events, got %d", len(loaded))
	}
}

func TestLoadFromVersionSemantics(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	for i := 0; i < 5; i++ {
		if err := store.Append(ctx, "agg", i, []eventsource.Event{{EventType: "E"}}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	// LoadFrom(3) must return versions 3,4,5 — not slice index 3 (which would be 4,5 only).
	from3, err := store.LoadFrom(ctx, "agg", 3)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(from3) != 3 {
		t.Fatalf("LoadFrom(3): expected 3 events, got %d", len(from3))
	}
	if from3[0].Version != 3 {
		t.Fatalf("LoadFrom(3): first version want 3 got %d", from3[0].Version)
	}

	from1, err := store.LoadFrom(ctx, "agg", 1)
	if err != nil {
		t.Fatalf("LoadFrom(1): %v", err)
	}
	if len(from1) != 5 {
		t.Fatalf("LoadFrom(1): expected full stream, got %d", len(from1))
	}

	from0, err := store.LoadFrom(ctx, "agg", 0)
	if err != nil {
		t.Fatalf("LoadFrom(0): %v", err)
	}
	if len(from0) != 5 {
		t.Fatalf("LoadFrom(0): expected full stream, got %d", len(from0))
	}

	pastEnd, err := store.LoadFrom(ctx, "agg", 100)
	if err != nil {
		t.Fatalf("LoadFrom(100): %v", err)
	}
	if len(pastEnd) != 0 {
		t.Fatalf("LoadFrom past end: expected 0, got %d", len(pastEnd))
	}
}

func TestLoadEmptyAndLoadAll(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	empty, err := store.Load(ctx, "missing")
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty slice, got %d", len(empty))
	}

	_ = store.Append(ctx, "a", 0, []eventsource.Event{{EventType: "X"}})
	_ = store.Append(ctx, "b", 0, []eventsource.Event{{EventType: "Y"}, {EventType: "Z"}})

	all, err := store.LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("LoadAll: expected 3, got %d", len(all))
	}
}

func TestConcurrentAppends(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()

	var conflictCount atomic.Int32
	var successCount atomic.Int32
	var wg sync.WaitGroup

	const n = 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			err := store.Append(ctx, "race", 0, []eventsource.Event{{EventType: "E"}})
			if err != nil {
				conflictCount.Add(1)
				return
			}
			successCount.Add(1)
		}()
	}
	wg.Wait()

	if successCount.Load() != 1 {
		t.Fatalf("expected exactly 1 successful first append, got %d", successCount.Load())
	}
	if conflictCount.Load() != n-1 {
		t.Fatalf("expected %d conflicts, got %d", n-1, conflictCount.Load())
	}
}

type testAgg struct {
	eventsource.BaseEventSourcedAggregate
	applied []string
}

func (a *testAgg) ApplyEvent(event eventsource.Event) error {
	a.applied = append(a.applied, event.EventType)
	return nil
}

func TestEventRepositorySaveAndLoadVersionBump(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	repo := eventsource.NewEventRepository(store)

	agg := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	if err := agg.RecordEvent("Created", map[string]string{"x": "1"}); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if err := agg.RecordEvent("Paid", map[string]string{"y": "2"}); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if agg.Version() != 0 {
		t.Fatalf("version before save want 0 got %d", agg.Version())
	}

	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if agg.Version() != 2 {
		t.Fatalf("version after save want 2 got %d", agg.Version())
	}
	if len(agg.GetUncommittedEvents()) != 0 {
		t.Fatal("uncommitted events should be cleared")
	}

	// Second save with new events uses bumped expected version.
	if err := agg.RecordEvent("Shipped", map[string]string{"z": "3"}); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if agg.Version() != 3 {
		t.Fatalf("version after second save want 3 got %d", agg.Version())
	}

	loaded := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	if err := repo.Load(ctx, loaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Version() != 3 {
		t.Fatalf("loaded version want 3 got %d", loaded.Version())
	}
	if len(loaded.applied) != 3 {
		t.Fatalf("expected 3 applied events, got %d", len(loaded.applied))
	}
}

func TestEventRepositorySaveConflict(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	repo := eventsource.NewEventRepository(store)

	a1 := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	_ = a1.RecordEvent("Created", map[string]string{})
	if err := repo.Save(ctx, a1); err != nil {
		t.Fatalf("Save a1: %v", err)
	}

	a2 := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	_ = a2.RecordEvent("AlsoCreated", map[string]string{})
	err := repo.Save(ctx, a2)
	if err == nil {
		t.Fatal("expected conflict on stale aggregate")
	}
}

func TestEventedStorePublishesToEventsBus(t *testing.T) {
	ctx := context.Background()
	bus := eventsmemory.New(events.Config{})
	defer bus.Close()

	var got []events.Event
	var mu sync.Mutex
	_, err := bus.Subscribe(ctx, "Order", func(ctx context.Context, e events.Event) error {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	inner := memory.NewEventStore()
	store := eventsource.NewEventedStore(inner, bus)

	err = store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order", ID: "e1"},
		{EventType: "order.paid", AggregateType: "Order", ID: "e2"},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Sync bus delivers immediately.
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(got))
	}
	if got[0].Type != "order.created" || got[1].Type != "order.paid" {
		t.Fatalf("unexpected types: %+v", got)
	}
	if got[0].Source != "pkg/enterprise/eventsource" {
		t.Fatalf("unexpected source %q", got[0].Source)
	}

	// Underlying store still has events.
	loaded, _ := store.Load(ctx, "o1")
	if len(loaded) != 2 {
		t.Fatalf("store load want 2 got %d", len(loaded))
	}
}

func TestEventedStoreNilBus(t *testing.T) {
	ctx := context.Background()
	store := eventsource.NewEventedStore(memory.NewEventStore(), nil)
	if err := store.Append(ctx, "o1", 0, []eventsource.Event{{EventType: "X"}}); err != nil {
		t.Fatalf("Append with nil bus: %v", err)
	}
}

func TestInstrumentedEventStore(t *testing.T) {
	ctx := context.Background()
	store := eventsource.NewInstrumentedEventStore(memory.NewEventStore())

	if err := store.Append(ctx, "o1", 0, []eventsource.Event{{EventType: "A"}}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	evts, err := store.Load(ctx, "o1")
	if err != nil || len(evts) != 1 {
		t.Fatalf("Load: %v len=%d", err, len(evts))
	}
	from, err := store.LoadFrom(ctx, "o1", 1)
	if err != nil || len(from) != 1 {
		t.Fatalf("LoadFrom: %v len=%d", err, len(from))
	}
	all, err := store.LoadAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("LoadAll: %v len=%d", err, len(all))
	}
}

func TestSnapshotStore(t *testing.T) {
	ctx := context.Background()
	ss := memory.NewSnapshotStore()

	snap, err := ss.Load(ctx, "missing")
	if err != nil || snap != nil {
		t.Fatalf("missing snapshot: snap=%v err=%v", snap, err)
	}

	err = ss.Save(ctx, eventsource.Snapshot{
		AggregateID:   "o1",
		AggregateType: "Order",
		Version:       5,
		Timestamp:     time.Now().UTC(),
		Data:          json.RawMessage(`{"total":10}`),
	})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := ss.Load(ctx, "o1")
	if err != nil || loaded == nil {
		t.Fatalf("Load: %v %v", loaded, err)
	}
	if loaded.Version != 5 {
		t.Fatalf("version want 5 got %d", loaded.Version)
	}
}

func TestAppendRequiresAggregateID(t *testing.T) {
	err := memory.NewEventStore().Append(context.Background(), "", 0, []eventsource.Event{{EventType: "A"}})
	if err == nil {
		t.Fatal("expected invalid argument")
	}
	var appErr *pkgerrors.AppError
	if !errors.As(err, &appErr) || appErr.Code != eventsource.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := memory.NewEventStore()
	if err := store.Append(ctx, "a", 0, []eventsource.Event{{EventType: "A"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
