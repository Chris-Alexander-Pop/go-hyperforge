package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/etlpipeline/server"
)

func TestCreateRunGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	cb, _ := json.Marshal(map[string]string{"name": "nightly"})
	cr, err := http.Post(ts.URL+"/v1/etl", "application/json", bytes.NewReader(cb))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var job map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&job)
	id, _ := job["id"].(string)

	rr, err := http.Post(ts.URL+"/v1/etl/"+id+"/run", "application/json", bytes.NewReader([]byte(`{"rows":[1,2]}`)))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer rr.Body.Close()
	json.NewDecoder(rr.Body).Decode(&job)
	if rr.StatusCode != http.StatusOK || job["status"] != "completed" {
		t.Fatalf("run failed: status=%d job=%v", rr.StatusCode, job)
	}

	gr, err := http.Get(ts.URL + "/v1/etl/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", gr.StatusCode)
	}
}

func TestGetMissing(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	gr, err := http.Get(ts.URL + "/v1/etl/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gr.Body.Close()
	if gr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", gr.StatusCode)
	}
}
