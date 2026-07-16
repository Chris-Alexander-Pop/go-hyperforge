package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/ratelimitersvc/server"
)

func TestCheckAllow(t *testing.T) {
	srv := server.New(server.Config{Port: "0", DefaultLimit: 2, DefaultPeriodSeconds: 60})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]interface{}{"key": "user:1", "limit": 2, "period_seconds": 60})
	var last map[string]interface{}
	for i := 0; i < 3; i++ {
		rr, err := http.Post(ts.URL+"/v1/ratelimits/check", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("check: %v", err)
		}
		json.NewDecoder(rr.Body).Decode(&last)
		rr.Body.Close()
		if rr.StatusCode != http.StatusOK {
			t.Fatalf("check status=%d", rr.StatusCode)
		}
	}
	if last["allowed"] != false {
		t.Fatalf("expected third check denied, got %v", last)
	}
}

func TestCheckMissingKey(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{})
	rr, err := http.Post(ts.URL+"/v1/ratelimits/check", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
