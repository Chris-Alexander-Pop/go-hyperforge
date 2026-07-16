package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/accesslogs/server"
)

func TestAppendListFilter(t *testing.T) {
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
		"user_id":  "user-1",
		"action":   "read",
		"resource": "doc-1",
		"outcome":  "success",
	})
	createResp, err := http.Post(ts.URL+"/v1/access-logs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("append status=%d", createResp.StatusCode)
	}

	other, _ := json.Marshal(map[string]string{
		"user_id": "user-2",
		"action":  "write",
	})
	otherResp, err := http.Post(ts.URL+"/v1/access-logs", "application/json", bytes.NewReader(other))
	if err != nil {
		t.Fatalf("append other: %v", err)
	}
	otherResp.Body.Close()

	listResp, err := http.Get(ts.URL + "/v1/access-logs?user_id=user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	var listed struct {
		Entries []struct {
			UserID string `json:"user_id"`
			Action string `json:"action"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(listed.Entries) != 1 || listed.Entries[0].UserID != "user-1" {
		t.Fatalf("unexpected list: %+v", listed)
	}
}

func TestAppendMissingUser(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"action": "read"})
	resp, err := http.Post(ts.URL+"/v1/access-logs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
