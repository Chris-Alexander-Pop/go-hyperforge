package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/promptengine/server"
)

func TestHealthPutGetRender(t *testing.T) {
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

	body, _ := json.Marshal(map[string]string{
		"name":     "greeting",
		"template": "Hello {{name}}",
	})
	putResp, err := http.Post(ts.URL+"/v1/prompts", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusCreated {
		t.Fatalf("put status=%d", putResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/prompts/greeting")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	rbody, _ := json.Marshal(map[string]interface{}{
		"vars": map[string]interface{}{"name": "Ada"},
	})
	renderResp, err := http.Post(ts.URL+"/v1/prompts/greeting/render", "application/json", bytes.NewReader(rbody))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	defer renderResp.Body.Close()
	if renderResp.StatusCode != http.StatusOK {
		t.Fatalf("render status=%d", renderResp.StatusCode)
	}
	var out struct {
		Rendered string `json:"rendered"`
	}
	if err := json.NewDecoder(renderResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Rendered != "Hello Ada" {
		t.Fatalf("unexpected render: %q", out.Rendered)
	}
}

func TestPutMissingName(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"template": "x"})
	resp, err := http.Post(ts.URL+"/v1/prompts", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetNotFound(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/prompts/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
