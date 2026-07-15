package warehouse_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/analytics/adapters/warehouse"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/bigdata"
)

type recordingClient struct {
	mu      sync.Mutex
	queries []recordedQuery
	closed  bool
}

type recordedQuery struct {
	Query string
	Args  []interface{}
}

func (c *recordingClient) Query(ctx context.Context, query string, args ...interface{}) (*bigdata.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]interface{}, len(args))
	copy(cp, args)
	c.queries = append(c.queries, recordedQuery{Query: query, Args: cp})
	return &bigdata.Result{Rows: nil, Metadata: map[string]interface{}{"source": "recording"}}, nil
}

func (c *recordingClient) Close() error {
	c.closed = true
	return nil
}

func (c *recordingClient) Queries() []recordedQuery {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]recordedQuery, len(c.queries))
	copy(out, c.queries)
	return out
}

func TestWarehouseSinkIngest(t *testing.T) {
	client := &recordingClient{}
	sink, err := warehouse.New(warehouse.Config{Client: client, Table: "analytics.events"})
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()

	ts := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	err = sink.Ingest(context.Background(),
		analytics.Event{Name: "page_view", UserID: "u1", Properties: map[string]any{"path": "/"}, Timestamp: ts},
		analytics.Event{Name: "click", UserID: "u1"},
	)
	if err != nil {
		t.Fatal(err)
	}

	qs := client.Queries()
	if len(qs) != 2 {
		t.Fatalf("queries=%d", len(qs))
	}
	if !strings.Contains(qs[0].Query, "INSERT INTO analytics.events") {
		t.Fatalf("query=%s", qs[0].Query)
	}
	if qs[0].Args[0] != "page_view" || qs[0].Args[1] != "u1" {
		t.Fatalf("args=%v", qs[0].Args)
	}
	if !strings.Contains(qs[0].Args[2].(string), "path") {
		t.Fatalf("props=%v", qs[0].Args[2])
	}
}

func TestWarehouseSinkRequiresConfig(t *testing.T) {
	if _, err := warehouse.New(warehouse.Config{}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := warehouse.New(warehouse.Config{Client: &recordingClient{}}); err == nil {
		t.Fatal("expected table error")
	}
}

func TestWarehouseSinkClosed(t *testing.T) {
	client := &recordingClient{}
	sink, err := warehouse.New(warehouse.Config{Client: client, Table: "events"})
	if err != nil {
		t.Fatal(err)
	}
	_ = sink.Close()
	if err := sink.Ingest(context.Background(), analytics.Event{Name: "x"}); err == nil {
		t.Fatal("expected ErrClosed")
	}
}
