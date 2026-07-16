package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/discovery/server"
)

func TestRegisterHeartbeatList(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]interface{}{"name": "api", "address": "10.0.0.1", "port": 8080})
	rr, err := http.Post(ts.URL+"/v1/services", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	var svc map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&svc)
	if rr.StatusCode != http.StatusCreated {
		t.Fatalf("register status=%d", rr.StatusCode)
	}
	id, _ := svc["id"].(string)

	hr, err := http.Post(ts.URL+"/v1/services/"+id+"/heartbeat", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	hr.Body.Close()
	if hr.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status=%d", hr.StatusCode)
	}

	lr, err := http.Get(ts.URL + "/v1/services?name=api")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer lr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(lr.Body).Decode(&out)
	svcs, _ := out["services"].([]interface{})
	if len(svcs) != 1 {
		t.Fatalf("expected 1 healthy service, got %v", out)
	}
}

func TestRegisterMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]interface{}{"address": "10.0.0.1", "port": 8080})
	rr, err := http.Post(ts.URL+"/v1/services", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
