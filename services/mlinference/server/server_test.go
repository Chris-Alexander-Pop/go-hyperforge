package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/mlinference/server"
)

func TestHealthAndPredict(t *testing.T) {
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
		"model_id": "demo-model",
		"input":    map[string]interface{}{"x": 1.5},
	})
	predictResp, err := http.Post(ts.URL+"/v1/inferences", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("predict: %v", err)
	}
	defer predictResp.Body.Close()
	if predictResp.StatusCode != http.StatusOK {
		t.Fatalf("predict status=%d", predictResp.StatusCode)
	}

	var out struct {
		Output map[string]interface{} `json:"output"`
	}
	if err := json.NewDecoder(predictResp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Output == nil || out.Output["model_id"] != "demo-model" {
		t.Fatalf("unexpected output: %+v", out.Output)
	}
}

func TestPredictMissingModelID(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"model_id": "",
		"input":    map[string]interface{}{"x": 1},
	})
	resp, err := http.Post(ts.URL+"/v1/inferences", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("predict: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
