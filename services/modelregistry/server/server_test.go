package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/modelregistry/server"
)

func TestRegisterAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"name": "ranker", "version": "1.0.0", "framework": "pytorch"})
	rr, err := http.Post(ts.URL+"/v1/models", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	var mv map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&mv)
	id, _ := mv["id"].(string)

	gr, err := http.Get(ts.URL + "/v1/models/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}
}

func TestRegisterMissingVersion(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"name": "ranker"})
	rr, err := http.Post(ts.URL+"/v1/models", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
