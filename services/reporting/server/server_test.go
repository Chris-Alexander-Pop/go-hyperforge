package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/reporting/server"
)

func TestCreateAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"name": "weekly", "type": "summary"})
	cr, err := http.Post(ts.URL+"/v1/reports", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var job map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&job)
	id, _ := job["id"].(string)

	gr, err := http.Get(ts.URL + "/v1/reports/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}
}

func TestGetMissing(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	gr, err := http.Get(ts.URL + "/v1/reports/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gr.Body.Close()
	if gr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", gr.StatusCode)
	}
}
