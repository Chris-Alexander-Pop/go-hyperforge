package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/agentorchestrator/server"
)

func TestHealthAndOrchestration(t *testing.T) {
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
		"goal":      "ship the feature",
		"agent_ids": []string{"researcher", "writer"},
	})
	createResp, err := http.Post(ts.URL+"/v1/orchestrations", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}

	var orch server.Orchestration
	if err := json.NewDecoder(createResp.Body).Decode(&orch); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if orch.ID == "" || orch.Status != server.OrchestrationCompleted {
		t.Fatalf("unexpected orchestration: %+v", orch)
	}
	if len(orch.Steps) != 2 || orch.Output == "" {
		t.Fatalf("unexpected steps/output: %+v", orch)
	}

	getResp, err := http.Get(ts.URL + "/v1/orchestrations/" + orch.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}
}

func TestCreateMissingGoal(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"goal": ""})
	resp, err := http.Post(ts.URL+"/v1/orchestrations", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
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

	resp, err := http.Get(ts.URL + "/v1/orchestrations/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
