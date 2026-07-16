package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/incidentmanager/server"
)

func TestCreateAckResolve(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"title": "db latency", "severity": "high"})
	res, err := http.Post(ts.URL+"/v1/incidents", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", res.StatusCode)
	}
	var created map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&created)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}

	ack, err := http.Post(ts.URL+"/v1/incidents/"+id+"/ack", "application/json", nil)
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	ack.Body.Close()
	if ack.StatusCode != http.StatusOK {
		t.Fatalf("ack status=%d", ack.StatusCode)
	}

	bad, err := http.Post(ts.URL+"/v1/incidents", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("bad: %v", err)
	}
	bad.Body.Close()
	if bad.StatusCode < 400 {
		t.Fatalf("expected 4xx for empty title, got %d", bad.StatusCode)
	}
}
