package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/analytics/server"
)

func TestTrackAndCount(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"name": "page_view", "user_id": "u1"})
	for i := 0; i < 2; i++ {
		tr, err := http.Post(ts.URL+"/v1/analytics/events", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("track: %v", err)
		}
		tr.Body.Close()
		if tr.StatusCode != http.StatusCreated {
			t.Fatalf("track status=%d", tr.StatusCode)
		}
	}
	qr, err := http.Get(ts.URL + "/v1/analytics/counts?name=page_view")
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	defer qr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(qr.Body).Decode(&out)
	if out["count"].(float64) != 2 {
		t.Fatalf("expected count 2, got %v", out)
	}
}

func TestTrackMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"user_id": "u1"})
	tr, err := http.Post(ts.URL+"/v1/analytics/events", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("track: %v", err)
	}
	defer tr.Body.Close()
	if tr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", tr.StatusCode)
	}
}
