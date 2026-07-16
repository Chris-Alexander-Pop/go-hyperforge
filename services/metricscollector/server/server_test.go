package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/metricscollector/server"
)

func TestIngestAndQuery(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", res.StatusCode)
	}

	body, _ := json.Marshal(map[string]interface{}{"name": "cpu", "value": 0.42, "labels": map[string]string{"host": "a"}})
	createResp, err := http.Post(ts.URL+"/v1/metrics", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("ingest status=%d", createResp.StatusCode)
	}

	qResp, err := http.Get(ts.URL + "/v1/metrics?name=cpu")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer qResp.Body.Close()
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("query status=%d", qResp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(qResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	pts, _ := out["datapoints"].([]interface{})
	if len(pts) != 1 {
		t.Fatalf("expected 1 datapoint, got %v", out)
	}
}

func TestQueryMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/metrics")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
