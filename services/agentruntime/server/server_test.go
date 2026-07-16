package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/agentruntime/server"
)

func TestHealthAndRun(t *testing.T) {
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
		"goal": "summarize the repo",
		"tools": []map[string]string{
			{"name": "echo", "description": "echo input"},
		},
	})
	createResp, err := http.Post(ts.URL+"/v1/agents/runs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}

	var run server.Run
	if err := json.NewDecoder(createResp.Body).Decode(&run); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if run.ID == "" || run.Status != server.RunStatusCompleted || run.Output == "" {
		t.Fatalf("unexpected run: %+v", run)
	}

	getResp, err := http.Get(ts.URL + "/v1/agents/runs/" + run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}
}

func TestCreateRunMissingGoal(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"goal": ""})
	resp, err := http.Post(ts.URL+"/v1/agents/runs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGetRunNotFound(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/agents/runs/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
