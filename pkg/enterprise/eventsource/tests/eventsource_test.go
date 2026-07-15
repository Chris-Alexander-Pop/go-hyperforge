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
	"github.com/chris-alexander-pop/go-hyperforge/pkg/messaging"
	msgmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/messaging/adapters/memory"
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

type snapAgg struct {
	eventsource.BaseEventSourcedAggregate
	total   int
	applied []string
}

type snapState struct {
	Total int `json:"total"`
}

func (a *snapAgg) ApplyEvent(event eventsource.Event) error {
	a.applied = append(a.applied, event.EventType)
	var payload struct {
		Amount int `json:"amount"`
	}
	if len(event.Data) > 0 {
		_ = json.Unmarshal(event.Data, &payload)
	}
	switch event.EventType {
	case "Created":
		a.total = payload.Amount
	case "Added":
		a.total += payload.Amount
	}
	return nil
}

func (a *snapAgg) SnapshotData() (json.RawMessage, error) {
	return json.Marshal(snapState{Total: a.total})
}

func (a *snapAgg) RestoreSnapshot(data json.RawMessage) error {
	var st snapState
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	a.total = st.Total
	return nil
}

func TestEventRepositoryLoadFromSnapshot(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	snaps := memory.NewSnapshotStore()
	repo := eventsource.NewEventRepositoryWithSnapshots(store, snaps)

	agg := &snapAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	for _, e := range []struct {
		typ string
		amt int
	}{
		{"Created", 10},
		{"Added", 5},
		{"Added", 3},
	} {
		if err := agg.RecordEvent(e.typ, map[string]int{"amount": e.amt}); err != nil {
			t.Fatalf("RecordEvent: %v", err)
		}
		evts := agg.GetUncommittedEvents()
		last := evts[len(evts)-1]
		if err := agg.ApplyEvent(last); err != nil {
			t.Fatalf("ApplyEvent: %v", err)
		}
	}
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if agg.Version() != 3 || agg.total != 18 {
		t.Fatalf("after save version=%d total=%d", agg.Version(), agg.total)
	}

	// Snapshot at version 2 (Created+Added=15); store still has event v3.
	if err := snaps.Save(ctx, eventsource.Snapshot{
		AggregateID:   "o1",
		AggregateType: "Order",
		Version:       2,
		Timestamp:     time.Now().UTC(),
		Data:          json.RawMessage(`{"total":15}`),
	}); err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}

	loaded := &snapAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	if err := repo.Load(ctx, loaded); err != nil {
		t.Fatalf("Load with snapshot: %v", err)
	}
	if loaded.Version() != 3 {
		t.Fatalf("loaded version want 3 got %d", loaded.Version())
	}
	if loaded.total != 18 {
		t.Fatalf("loaded total want 18 got %d", loaded.total)
	}
	// Only events after the snapshot (version 3) should have been applied.
	if len(loaded.applied) != 1 || loaded.applied[0] != "Added" {
		t.Fatalf("expected only post-snapshot event applied, got %v", loaded.applied)
	}

	// SaveSnapshot round-trip at current version.
	if err := repo.SaveSnapshot(ctx, loaded); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	snap, err := snaps.Load(ctx, "o1")
	if err != nil || snap == nil {
		t.Fatalf("Load snapshot: %v %v", snap, err)
	}
	if snap.Version != 3 || string(snap.Data) != `{"total":18}` {
		t.Fatalf("snapshot=%+v data=%s", snap, snap.Data)
	}
}

func TestEventRepositoryLoadWithoutSnapshotFallsBack(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	repo := eventsource.NewEventRepositoryWithSnapshots(store, memory.NewSnapshotStore())

	agg := &snapAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	_ = agg.RecordEvent("Created", map[string]int{"amount": 7})
	_ = agg.RecordEvent("Added", map[string]int{"amount": 1})
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := &snapAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	if err := repo.Load(ctx, loaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Version() != 2 || loaded.total != 8 {
		t.Fatalf("version=%d total=%d", loaded.Version(), loaded.total)
	}
	if len(loaded.applied) != 2 {
		t.Fatalf("expected full replay, got %v", loaded.applied)
	}
}

func TestEventRepositoryPlainAggregateIgnoresSnapshots(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	snaps := memory.NewSnapshotStore()
	repo := eventsource.NewEventRepositoryWithSnapshots(store, snaps)

	agg := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	_ = agg.RecordEvent("Created", map[string]string{})
	_ = agg.RecordEvent("Paid", map[string]string{})
	if err := repo.Save(ctx, agg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	_ = snaps.Save(ctx, eventsource.Snapshot{
		AggregateID: "o1", AggregateType: "Order", Version: 1,
		Data: json.RawMessage(`{}`),
	})

	loaded := &testAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	if err := repo.Load(ctx, loaded); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.applied) != 2 {
		t.Fatalf("plain aggregate should full-replay, got %v", loaded.applied)
	}
}

func TestSaveSnapshotRequiresStore(t *testing.T) {
	repo := eventsource.NewEventRepository(memory.NewEventStore())
	agg := &snapAgg{BaseEventSourcedAggregate: eventsource.NewBaseEventSourcedAggregate("o1", "Order")}
	err := repo.SaveSnapshot(context.Background(), agg)
	if err == nil {
		t.Fatal("expected error without snapshot store")
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

type countingProjector struct {
	mu    sync.Mutex
	types []string
	seen  []string
}

func (p *countingProjector) EventTypes() []string { return p.types }

func (p *countingProjector) Project(ctx context.Context, event interface{}) error {
	ev, ok := event.(eventsource.Event)
	if !ok {
		return errors.New("unexpected event type")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seen = append(p.seen, ev.EventType)
	return nil
}

func (p *countingProjector) Seen() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.seen))
	copy(out, p.seen)
	return out
}

func TestProjectionRunnerCatchUpAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	_ = store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order"},
		{EventType: "order.paid", AggregateType: "Order"},
		{EventType: "order.shipped", AggregateType: "Order"},
	})

	proj := &countingProjector{types: []string{"order.created", "order.paid"}}
	cps := memory.NewCheckpointStore()
	runner := eventsource.NewProjectionRunner(store, cps, proj, eventsource.ProjectionConfig{Name: "orders-rm"})

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(proj.Seen()) != 2 {
		t.Fatalf("expected 2 projected events, got %v", proj.Seen())
	}
	cp, err := cps.Load(ctx, "orders-rm")
	if err != nil {
		t.Fatalf("Load checkpoint: %v", err)
	}
	if cp.Position != 3 {
		t.Fatalf("checkpoint position want 3 got %d", cp.Position)
	}

	// Second run should not re-project.
	if err := runner.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce 2: %v", err)
	}
	if len(proj.Seen()) != 2 {
		t.Fatalf("expected no re-projection, got %v", proj.Seen())
	}

	// New events continue from checkpoint.
	_ = store.Append(ctx, "o1", 3, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order"},
	})
	if err := runner.RunOnce(ctx); err != nil {
		t.Fatalf("RunOnce 3: %v", err)
	}
	if len(proj.Seen()) != 3 {
		t.Fatalf("expected 3 projected events, got %v", proj.Seen())
	}
}

type recordingMetrics struct {
	mu      sync.Mutex
	batches int
	errors  int
	idle    int
}

func (m *recordingMetrics) OnBatch(name string, applied int, advanced int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.batches++
}
func (m *recordingMetrics) OnError(name string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
}
func (m *recordingMetrics) OnCatchUpIdle(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idle++
}
func (m *recordingMetrics) Batches() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.batches
}
func (m *recordingMetrics) Errors() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errors
}

type failOnceProjector struct {
	mu     sync.Mutex
	types  []string
	calls  int
	failAt int
	seen   []string
}

func (p *failOnceProjector) EventTypes() []string { return p.types }
func (p *failOnceProjector) Project(ctx context.Context, event interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	ev := event.(eventsource.Event)
	if p.calls == p.failAt {
		return errors.New("boom")
	}
	p.seen = append(p.seen, ev.EventType)
	return nil
}
func (p *failOnceProjector) Seen() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.seen))
	copy(out, p.seen)
	return out
}

func TestProjectionRunnerResetAndInstrumented(t *testing.T) {
	ctx := context.Background()
	store := memory.NewEventStore()
	_ = store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order"},
	})
	metrics := &recordingMetrics{}
	proj := &countingProjector{types: []string{"order.created"}}
	cps := memory.NewCheckpointStore()
	runner := eventsource.NewProjectionRunner(store, cps, proj, eventsource.ProjectionConfig{
		Name: "rm", Metrics: metrics,
	})
	inst := eventsource.NewInstrumentedProjectionRunner(runner)

	if err := inst.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if metrics.Batches() != 1 {
		t.Fatalf("batches=%d", metrics.Batches())
	}
	if err := inst.ResetCheckpoint(ctx); err != nil {
		t.Fatal(err)
	}
	cp, err := inst.Checkpoint(ctx)
	if err != nil || cp.Position != 0 {
		t.Fatalf("checkpoint after reset: %+v err=%v", cp, err)
	}
	if err := inst.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(proj.Seen()) != 2 {
		t.Fatalf("expected re-projection after reset, got %v", proj.Seen())
	}
}

func TestProjectionRunnerRunBackoffAndConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := memory.NewEventStore()
	_ = store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order"},
	})
	metrics := &recordingMetrics{}
	proj := &failOnceProjector{types: []string{"order.created"}, failAt: 1}
	cps := memory.NewCheckpointStore()

	runner, err := eventsource.NewProjectionFromConfig(eventsource.Config{
		ProjectionName: "cfg-rm",
		PollInterval:   5 * time.Millisecond,
		InitialBackoff: 5 * time.Millisecond,
		MaxBackoff:     20 * time.Millisecond,
	}, eventsource.ProjectionParts{Store: store, Checkpoints: cps, Projector: proj})
	if err != nil {
		t.Fatal(err)
	}
	if runner.Name() != "cfg-rm" {
		t.Fatalf("name=%s", runner.Name())
	}
	// Rebuild with metrics hooks for the continuous Run backoff assertion.
	runner = eventsource.NewProjectionRunner(store, cps, proj, eventsource.ProjectionConfig{
		Name: "cfg-rm", PollInterval: 5 * time.Millisecond,
		InitialBackoff: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond,
		Metrics: metrics,
	})

	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for successful projection after backoff")
		case <-time.After(10 * time.Millisecond):
			if len(proj.Seen()) >= 1 && metrics.Errors() >= 1 {
				cancel()
				<-done
				return
			}
		}
	}
}

func TestContinuousProjectorOutboxAndEventStore(t *testing.T) {
	ctx := context.Background()
	broker := msgmemory.New(msgmemory.Config{BufferSize: 8})
	defer broker.Close()
	producer, err := broker.Producer("Order")
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := broker.Consumer("Order", "proj")
	if err != nil {
		t.Fatal(err)
	}

	proj := &countingProjector{types: []string{"order.created"}}
	cp, err := eventsource.NewContinuousProjector(eventsource.ContinuousProjectorConfig{
		ProjectionConfig: eventsource.ProjectionConfig{
			Name: "outbox-rm", PollInterval: 5 * time.Millisecond,
			InitialBackoff: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond,
		},
		Projector: proj,
		Consumer:  consumer,
	})
	if err != nil {
		t.Fatal(err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- cp.Run(runCtx) }()

	store := eventsource.NewEventedStoreWithOutbox(memory.NewEventStore(), nil, producer)
	if err := store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order", ID: "e1", Data: json.RawMessage(`{"id":"1"}`)},
		{EventType: "order.ignored", AggregateType: "Order", ID: "e2"},
	}); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for outbox projection")
		case <-time.After(10 * time.Millisecond):
			seen := proj.Seen()
			if len(seen) >= 1 {
				cancel()
				<-done
				if seen[0] != "order.created" {
					t.Fatalf("seen=%v", seen)
				}
				goto eventStoreMode
			}
		}
	}

eventStoreMode:
	store2 := memory.NewEventStore()
	_ = store2.Append(ctx, "o1", 0, []eventsource.Event{{EventType: "order.created"}})
	proj2 := &countingProjector{types: []string{"order.created"}}
	cps := memory.NewCheckpointStore()
	cp2, err := eventsource.NewContinuousProjector(eventsource.ContinuousProjectorConfig{
		ProjectionConfig: eventsource.ProjectionConfig{Name: "es-rm", PollInterval: 5 * time.Millisecond},
		Store:            store2,
		Checkpoints:      cps,
		Projector:        proj2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cp2.Runner() == nil {
		t.Fatal("expected ProjectionRunner")
	}
	esCtx, esCancel := context.WithCancel(ctx)
	defer esCancel()
	esDone := make(chan error, 1)
	go func() { esDone <- cp2.Run(esCtx) }()
	deadline = time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for eventstore continuous projection")
		case <-time.After(10 * time.Millisecond):
			if len(proj2.Seen()) >= 1 {
				esCancel()
				<-esDone
				return
			}
		}
	}
}

func TestContinuousProjectorRequiresSource(t *testing.T) {
	_, err := eventsource.NewContinuousProjector(eventsource.ContinuousProjectorConfig{
		Projector: &countingProjector{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEventedStoreWithOutbox(t *testing.T) {
	ctx := context.Background()
	broker := msgmemory.New(msgmemory.Config{BufferSize: 8})
	defer broker.Close()
	producer, err := broker.Producer("Order")
	if err != nil {
		t.Fatalf("Producer: %v", err)
	}
	consumer, err := broker.Consumer("Order", "g1")
	if err != nil {
		t.Fatalf("Consumer: %v", err)
	}

	got := make(chan []byte, 2)
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	go func() {
		_ = consumer.Consume(cctx, func(ctx context.Context, msg *messaging.Message) error {
			got <- msg.Payload
			return nil
		})
	}()

	store := eventsource.NewEventedStoreWithOutbox(memory.NewEventStore(), nil, producer)
	if err := store.Append(ctx, "o1", 0, []eventsource.Event{
		{EventType: "order.created", AggregateType: "Order", ID: "e1"},
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	select {
	case payload := <-got:
		var env events.OutboxPayload
		if err := json.Unmarshal(payload, &env); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if env.Type != "order.created" || env.Topic != "Order" {
			t.Fatalf("envelope=%+v", env)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for messaging outbox")
	}
}
