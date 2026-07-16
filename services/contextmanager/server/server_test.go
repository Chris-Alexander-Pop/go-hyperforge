package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/services/contextmanager/server"
)

func TestHealthCreateAppendList(t *testing.T) {
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

	createResp, err := http.Post(ts.URL+"/v1/contexts", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}
	var session server.Session
	if err := json.NewDecoder(createResp.Body).Decode(&session); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected session id")
	}

	body, _ := json.Marshal(map[string]string{
		"role":    string(llm.RoleUser),
		"content": "hello",
	})
	appendResp, err := http.Post(ts.URL+"/v1/contexts/"+session.ID+"/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	defer appendResp.Body.Close()
	if appendResp.StatusCode != http.StatusCreated {
		t.Fatalf("append status=%d", appendResp.StatusCode)
	}

	listResp, err := http.Get(ts.URL + "/v1/contexts/" + session.ID + "/messages")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listResp.StatusCode)
	}
	var msgs []llm.Message
	if err := json.NewDecoder(listResp.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Content != "hello" {
		t.Fatalf("unexpected messages: %+v", msgs)
	}
}

func TestAppendMissingContent(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	createResp, err := http.Post(ts.URL+"/v1/contexts", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	var session server.Session
	if err := json.NewDecoder(createResp.Body).Decode(&session); err != nil {
		t.Fatalf("decode: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"role": "user", "content": ""})
	resp, err := http.Post(ts.URL+"/v1/contexts/"+session.ID+"/messages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestContextNotFound(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/contexts/missing/messages")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
