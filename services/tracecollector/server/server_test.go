package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/tracecollector/server"
)

func TestIngestAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"trace_id": "t1", "name": "GET /api"})
	resp, err := http.Post(ts.URL+"/v1/traces", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("ingest status=%d", resp.StatusCode)
	}

	g, err := http.Get(ts.URL + "/v1/traces/t1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer g.Body.Close()
	if g.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", g.StatusCode)
	}
}

func TestGetMissing(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	g, err := http.Get(ts.URL + "/v1/traces/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer g.Body.Close()
	if g.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", g.StatusCode)
	}
}
