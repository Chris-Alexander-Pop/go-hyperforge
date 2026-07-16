package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/schemaregistry/server"
)

func TestRegisterAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"subject": "order", "format": "json", "schema": `{"type":"object"}`})
	rr, err := http.Post(ts.URL+"/v1/schemas", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	var sv map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&sv)
	id, _ := sv["id"].(string)
	if rr.StatusCode != http.StatusCreated {
		t.Fatalf("register status=%d", rr.StatusCode)
	}

	gr, err := http.Get(ts.URL + "/v1/schemas/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}

	sr, err := http.Get(ts.URL + "/v1/schemas?subject=order")
	if err != nil {
		t.Fatalf("by subject: %v", err)
	}
	sr.Body.Close()
	if sr.StatusCode != http.StatusOK {
		t.Fatalf("subject=%d", sr.StatusCode)
	}
}

func TestRegisterMissingSchema(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"subject": "order"})
	rr, err := http.Post(ts.URL+"/v1/schemas", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
