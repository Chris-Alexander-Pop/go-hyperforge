package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/scheduledjobs/server"
)

func TestCreateAndRun(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"name": "nightly", "schedule": "once"})
	cr, err := http.Post(ts.URL+"/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", cr.StatusCode)
	}

	rr, err := http.Post(ts.URL+"/v1/jobs/nightly/run", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer rr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&out)
	if rr.StatusCode != http.StatusOK {
		t.Fatalf("run status=%d out=%v", rr.StatusCode, out)
	}
	if out["runs"].(float64) < 1 {
		t.Fatalf("expected runs >= 1, got %v", out)
	}
}

func TestCreateMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"schedule": "once"})
	cr, err := http.Post(ts.URL+"/v1/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", cr.StatusCode)
	}
}
