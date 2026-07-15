package analytics_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/memory"
)

func TestMemorySinkIngest(t *testing.T) {
	sink := memory.NewSink()
	defer sink.Close()
	ctx := context.Background()

	err := sink.Ingest(ctx,
		analytics.Event{Name: "page_view", UserID: "u1", Properties: map[string]any{"path": "/"}},
		analytics.Event{Name: "click", UserID: "u1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("len=%d", len(events))
	}
	if events[0].Name != "page_view" || events[0].Timestamp.IsZero() {
		t.Fatalf("event0=%+v", events[0])
	}

	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	if err := sink.Ingest(ctx, analytics.Event{Name: "x"}); err == nil {
		t.Fatal("expected ErrClosed")
	}
}

func TestWindowKey(t *testing.T) {
	ts := time.Date(2026, 7, 15, 5, 34, 0, 0, time.UTC)
	key := analytics.WindowKey("visitors", ts, time.Hour)
	if !strings.HasPrefix(key, "visitors:") {
		t.Fatalf("key=%s", key)
	}
	if key != "visitors:2026-07-15T05" {
		t.Fatalf("key=%s", key)
	}

	day := analytics.WindowKey("visitors", ts, 24*time.Hour)
	if day != "visitors:2026-07-15" {
		t.Fatalf("day=%s", day)
	}
}

func TestWindowedUniqueness(t *testing.T) {
	tracker, err := memory.New(analytics.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer tracker.Close()

	w := analytics.NewWindowedUniqueness(tracker, "visitors", time.Hour)
	ts := time.Date(2026, 7, 15, 5, 10, 0, 0, time.UTC)
	ctx := context.Background()

	if err := w.Add(ctx, "u1", ts); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(ctx, "u2", ts); err != nil {
		t.Fatal(err)
	}
	if err := w.Add(ctx, "u1", ts); err != nil {
		t.Fatal(err)
	}

	n, err := w.Count(ctx, ts)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("count=%d", n)
	}

	// Different hour bucket should be empty.
	other := ts.Add(2 * time.Hour)
	n2, err := w.Count(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 0 {
		t.Fatalf("other window count=%d", n2)
	}
}
