package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/featureflag/server"
)

func TestCreateAndEvaluate(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	cb, _ := json.Marshal(map[string]interface{}{"key": "new-ui", "enabled": true, "percentage": 100})
	cr, err := http.Post(ts.URL+"/v1/flags", "application/json", bytes.NewReader(cb))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	cr.Body.Close()
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", cr.StatusCode)
	}

	eb, _ := json.Marshal(map[string]string{"flag": "new-ui", "user_id": "u1"})
	er, err := http.Post(ts.URL+"/v1/flags/evaluate", "application/json", bytes.NewReader(eb))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer er.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(er.Body).Decode(&out)
	if er.StatusCode != http.StatusOK || out["enabled"] != true {
		t.Fatalf("evaluate failed: status=%d out=%v", er.StatusCode, out)
	}
}

func TestEvaluateMissingFlag(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	eb, _ := json.Marshal(map[string]string{"flag": "missing", "user_id": "u1"})
	er, err := http.Post(ts.URL+"/v1/flags/evaluate", "application/json", bytes.NewReader(eb))
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer er.Body.Close()
	if er.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", er.StatusCode)
	}
}
