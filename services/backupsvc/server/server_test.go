package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/backupsvc/server"
)

func TestCreateListRestore(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"source": "db/main"})
	cr, err := http.Post(ts.URL+"/v1/backups", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var b map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&b)
	id, _ := b["id"].(string)

	lr, _ := http.Get(ts.URL + "/v1/backups")
	lr.Body.Close()

	rr, err := http.Post(ts.URL+"/v1/backups/"+id+"/restore", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusOK {
		t.Fatalf("restore status=%d", rr.StatusCode)
	}
}

func TestRestoreMissing(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	rr, err := http.Post(ts.URL+"/v1/backups/missing/restore", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.StatusCode)
	}
}
