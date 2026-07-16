package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/kycverification/server"
)

func TestSubmitApproveGet(t *testing.T) {
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
		"subject_id": "user-1",
		"full_name":  "Ada Lovelace",
		"document":   "passport-1",
	})
	createResp, err := http.Post(ts.URL+"/v1/kyc/applications", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("submit status=%d", createResp.StatusCode)
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ID == "" || created.Status != "pending" {
		t.Fatalf("unexpected create: %+v", created)
	}

	approveResp, err := http.Post(ts.URL+"/v1/kyc/applications/"+created.ID+"/approve", "application/json", nil)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	approveResp.Body.Close()
	if approveResp.StatusCode != http.StatusOK {
		t.Fatalf("approve status=%d", approveResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/kyc/applications/" + created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	var got struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Status != "approved" {
		t.Fatalf("status=%q", got.Status)
	}
}

func TestSubmitMissingSubject(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"full_name": "x"})
	resp, err := http.Post(ts.URL+"/v1/kyc/applications", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
