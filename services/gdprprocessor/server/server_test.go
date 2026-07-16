package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/gdprprocessor/server"
)

func TestCreateCompleteGet(t *testing.T) {
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
		"type":       "erasure",
		"subject_id": "user-1",
	})
	createResp, err := http.Post(ts.URL+"/v1/gdpr/requests", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.Status != "pending" {
		t.Fatalf("status=%q", created.Status)
	}

	completeResp, err := http.Post(ts.URL+"/v1/gdpr/requests/"+created.ID+"/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	completeResp.Body.Close()
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("complete status=%d", completeResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/gdpr/requests/" + created.ID)
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
	if got.Status != "completed" {
		t.Fatalf("status=%q", got.Status)
	}
}

func TestCreateInvalidType(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"type": "unknown", "subject_id": "u"})
	resp, err := http.Post(ts.URL+"/v1/gdpr/requests", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
