package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/datacatalog/server"
)

func TestRegisterListGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"name": "orders", "description": "order facts"})
	rr, err := http.Post(ts.URL+"/v1/catalogs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	var ds map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&ds)
	id, _ := ds["id"].(string)

	lr, _ := http.Get(ts.URL + "/v1/catalogs")
	lr.Body.Close()
	if lr.StatusCode != http.StatusOK {
		t.Fatalf("list=%d", lr.StatusCode)
	}
	gr, err := http.Get(ts.URL + "/v1/catalogs/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get=%d", gr.StatusCode)
	}
}

func TestRegisterMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"description": "x"})
	rr, err := http.Post(ts.URL+"/v1/catalogs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
