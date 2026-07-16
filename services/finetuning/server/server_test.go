package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/finetuning/server"
)

func TestCreateAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"model": "base-llm", "dataset": "ds-1"})
	cr, err := http.Post(ts.URL+"/v1/finetunes", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var job map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&job)
	id, _ := job["id"].(string)
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create=%d", cr.StatusCode)
	}

	gr, err := http.Get(ts.URL + "/v1/finetunes/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}
}

func TestCreateMissingModel(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"dataset": "ds-1"})
	cr, err := http.Post(ts.URL+"/v1/finetunes", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", cr.StatusCode)
	}
}
