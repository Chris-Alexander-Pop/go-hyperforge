package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/embeddingsvc/server"
)

func TestHealthAndEmbed(t *testing.T) {
	srv := server.New(server.Config{Port: "0", Dimension: 8})
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

	body, _ := json.Marshal(map[string]interface{}{"texts": []string{"a", "b"}})
	embedResp, err := http.Post(ts.URL+"/v1/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	defer embedResp.Body.Close()
	if embedResp.StatusCode != http.StatusOK {
		t.Fatalf("embed status=%d", embedResp.StatusCode)
	}

	var out struct {
		Vectors   [][]float32 `json:"vectors"`
		Dimension int         `json:"dimension"`
	}
	if err := json.NewDecoder(embedResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Dimension != 8 || len(out.Vectors) != 2 || len(out.Vectors[0]) != 8 {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestEmbedEmptyTexts(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"texts": []string{}})
	resp, err := http.Post(ts.URL+"/v1/embeddings", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
