package server_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/secretmanager/server"
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

	body, _ := json.Marshal(map[string]string{"name": "db-pass", "value": "s3cret"})
	setResp, err := http.Post(ts.URL+"/v1/secrets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	setResp.Body.Close()
	if setResp.StatusCode != http.StatusOK {
		t.Fatalf("set status=%d", setResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/secrets/db-pass")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}
	var out struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Value != "s3cret" {
		t.Fatalf("value=%q", out.Value)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/secrets/db-pass", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}

	miss, err := http.Get(ts.URL + "/v1/secrets/db-pass")
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	io.Copy(io.Discard, miss.Body)
	miss.Body.Close()
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", miss.StatusCode)
	}
}

func TestSetMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"value": "x"})
	resp, err := http.Post(ts.URL+"/v1/secrets", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
