package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/archival/server"
)

func TestArchiveListGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"object_key": "logs/2024", "tier": "glacier"})
	ar, err := http.Post(ts.URL+"/v1/archives", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	defer ar.Body.Close()
	var obj map[string]interface{}
	json.NewDecoder(ar.Body).Decode(&obj)
	id, _ := obj["id"].(string)

	lr, _ := http.Get(ts.URL + "/v1/archives")
	lr.Body.Close()
	gr, err := http.Get(ts.URL + "/v1/archives/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}
}

func TestArchiveMissingKey(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"tier": "cold"})
	ar, err := http.Post(ts.URL+"/v1/archives", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	defer ar.Body.Close()
	if ar.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ar.StatusCode)
	}
}
