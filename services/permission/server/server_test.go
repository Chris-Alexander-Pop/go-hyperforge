package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/permission/server"
)

func TestGrantCheckRevoke(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"subject": "editor", "resource": "docs", "action": "write"})
	gr, err := http.Post(ts.URL+"/v1/permissions/grant", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusCreated {
		t.Fatalf("grant=%d", gr.StatusCode)
	}

	cr, err := http.Post(ts.URL+"/v1/permissions/check", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer cr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&out)
	if out["allowed"] != true {
		t.Fatalf("expected allowed, got %v", out)
	}

	rr, err := http.Post(ts.URL+"/v1/permissions/revoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	rr.Body.Close()
	if rr.StatusCode != http.StatusOK {
		t.Fatalf("revoke=%d", rr.StatusCode)
	}

	cr2, err := http.Post(ts.URL+"/v1/permissions/check", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("check2: %v", err)
	}
	defer cr2.Body.Close()
	json.NewDecoder(cr2.Body).Decode(&out)
	if out["allowed"] != false {
		t.Fatalf("expected denied after revoke, got %v", out)
	}
}

func TestCheckMissingFields(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"subject": "editor"})
	cr, err := http.Post(ts.URL+"/v1/permissions/check", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", cr.StatusCode)
	}
}
