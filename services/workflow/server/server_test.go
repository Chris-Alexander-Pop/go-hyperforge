package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/workflow"
	"github.com/chris-alexander-pop/go-hyperforge/services/workflow/server"
)

func TestHealthRegisterStartGet(t *testing.T) {
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

	defBody, _ := json.Marshal(map[string]interface{}{
		"id":   "wf-echo",
		"name": "echo",
	})
	defResp, err := http.Post(ts.URL+"/v1/workflows/definitions", "application/json", bytes.NewReader(defBody))
	if err != nil {
		t.Fatalf("definitions: %v", err)
	}
	defer defResp.Body.Close()
	if defResp.StatusCode != http.StatusCreated {
		t.Fatalf("definitions status=%d", defResp.StatusCode)
	}

	startBody, _ := json.Marshal(map[string]interface{}{
		"workflow_id": "wf-echo",
		"input":       map[string]string{"hello": "world"},
	})
	startResp, err := http.Post(ts.URL+"/v1/workflows/start", "application/json", bytes.NewReader(startBody))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer startResp.Body.Close()
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("start status=%d", startResp.StatusCode)
	}

	var exec workflow.Execution
	if err := json.NewDecoder(startResp.Body).Decode(&exec); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	if exec.ID == "" {
		t.Fatal("expected execution id")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		getResp, err := http.Get(ts.URL + "/v1/workflows/executions/" + exec.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		var got workflow.Execution
		_ = json.NewDecoder(getResp.Body).Decode(&got)
		getResp.Body.Close()
		if getResp.StatusCode != http.StatusOK {
			t.Fatalf("get status=%d", getResp.StatusCode)
		}
		if got.Status == workflow.StatusCompleted || got.Status == workflow.StatusFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("execution did not finish: status=%s", got.Status)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestStartMissingWorkflowID(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(ts.URL+"/v1/workflows/start", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
