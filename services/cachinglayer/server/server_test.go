package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/cachinglayer/server"
)

func TestHealthSetGetDelete(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", res.StatusCode)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"key":         "greeting",
		"value":       "hello",
		"ttl_seconds": 60,
	})
	setResp, err := http.Post(ts.URL+"/v1/caches", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	defer setResp.Body.Close()
	if setResp.StatusCode != http.StatusCreated {
		t.Fatalf("set status=%d", setResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/caches/greeting")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}
	var got struct {
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Value != "hello" {
		t.Fatalf("value=%v", got.Value)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/caches/greeting", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}
}

func TestSetMissingKey(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"value": "x"})
	resp, err := http.Post(ts.URL+"/v1/caches", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
