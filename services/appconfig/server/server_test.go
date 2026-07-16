package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/appconfig/server"
)

func TestSetAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]interface{}{"key": "theme", "value": "dark"})
	sr, err := http.Post(ts.URL+"/v1/configs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	sr.Body.Close()
	if sr.StatusCode != http.StatusOK {
		t.Fatalf("set=%d", sr.StatusCode)
	}

	gr, err := http.Get(ts.URL + "/v1/configs/theme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(gr.Body).Decode(&out)
	if gr.StatusCode != http.StatusOK || out["value"] != "dark" {
		t.Fatalf("get failed: %d %v", gr.StatusCode, out)
	}
}

func TestGetMissing(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	gr, err := http.Get(ts.URL + "/v1/configs/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gr.Body.Close()
	if gr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", gr.StatusCode)
	}
}
