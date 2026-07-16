package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/compliance/server"
)

func TestRecordListGet(t *testing.T) {
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
		"policy":     "aml",
		"subject_id": "user-1",
		"result":     "pass",
	})
	createResp, err := http.Post(ts.URL+"/v1/compliance/checks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("record status=%d", createResp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	listResp, err := http.Get(ts.URL + "/v1/compliance/checks")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	var listed struct {
		Checks []struct {
			ID string `json:"id"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Checks) != 1 || listed.Checks[0].ID != created.ID {
		t.Fatalf("unexpected list: %+v", listed)
	}

	getResp, err := http.Get(ts.URL + "/v1/compliance/checks/" + created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}
}

func TestRecordMissingPolicy(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"subject_id": "u", "result": "pass"})
	resp, err := http.Post(ts.URL+"/v1/compliance/checks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
