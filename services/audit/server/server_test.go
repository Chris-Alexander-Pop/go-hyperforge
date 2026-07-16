package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/audit/server"
)

func TestAppendAndQuery(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"actor_id": "u1", "action": "update", "resource_id": "doc-1", "outcome": "success"})
	ar, err := http.Post(ts.URL+"/v1/audits", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	ar.Body.Close()
	if ar.StatusCode != http.StatusCreated {
		t.Fatalf("append=%d", ar.StatusCode)
	}

	qr, err := http.Get(ts.URL + "/v1/audits?actor_id=u1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer qr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(qr.Body).Decode(&out)
	items, _ := out["audits"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 audit, got %v", out)
	}
}

func TestAppendMissingActor(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"action": "update"})
	ar, err := http.Post(ts.URL+"/v1/audits", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	defer ar.Body.Close()
	if ar.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", ar.StatusCode)
	}
}
