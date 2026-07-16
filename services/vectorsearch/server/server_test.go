package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/services/vectorsearch/server"
)

func TestHealthUpsertQuery(t *testing.T) {
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

	body, _ := json.Marshal(map[string]interface{}{
		"id":       "doc-1",
		"vector":   []float32{1, 0, 0},
		"metadata": map[string]interface{}{"source": "test"},
	})
	upsertResp, err := http.Post(ts.URL+"/v1/vectors", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	defer upsertResp.Body.Close()
	if upsertResp.StatusCode != http.StatusOK {
		t.Fatalf("upsert status=%d", upsertResp.StatusCode)
	}

	qbody, _ := json.Marshal(map[string]interface{}{
		"vector": []float32{1, 0, 0},
		"top_k":  1,
	})
	queryResp, err := http.Post(ts.URL+"/v1/vectors/query", "application/json", bytes.NewReader(qbody))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer queryResp.Body.Close()
	if queryResp.StatusCode != http.StatusOK {
		t.Fatalf("query status=%d", queryResp.StatusCode)
	}

	var out struct {
		Matches []vector.Result `json:"matches"`
	}
	if err := json.NewDecoder(queryResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Matches) != 1 || out.Matches[0].ID != "doc-1" {
		t.Fatalf("unexpected matches: %+v", out.Matches)
	}
}

func TestUpsertMissingVector(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"id": "x", "vector": []float32{}})
	resp, err := http.Post(ts.URL+"/v1/vectors", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
