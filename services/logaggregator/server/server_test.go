package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/logaggregator/server"
)

func TestIngestAndFilter(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz=%d", res.StatusCode)
	}

	for _, body := range []map[string]string{
		{"level": "info", "message": "hello"},
		{"level": "error", "message": "boom"},
	} {
		b, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+"/v1/logs", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatalf("ingest: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("ingest status=%d", resp.StatusCode)
		}
	}

	q, err := http.Get(ts.URL + "/v1/logs?level=error")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer q.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(q.Body).Decode(&out)
	logs, _ := out["logs"].([]interface{})
	if len(logs) != 1 {
		t.Fatalf("expected 1 error log, got %v", out)
	}
}

func TestIngestMissingMessage(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	b, _ := json.Marshal(map[string]string{"level": "info"})
	resp, err := http.Post(ts.URL+"/v1/logs", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
