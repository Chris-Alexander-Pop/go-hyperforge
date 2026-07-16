package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/ai/genai/llm"
	"github.com/chris-alexander-pop/go-hyperforge/services/llmgateway/server"
)

func TestHealthAndChat(t *testing.T) {
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
		"messages": []map[string]string{
			{"role": string(llm.RoleUser), "content": "hello"},
		},
	})
	chatResp, err := http.Post(ts.URL+"/v1/llm-requests/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	defer chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("chat status=%d", chatResp.StatusCode)
	}

	var gen llm.Generation
	if err := json.NewDecoder(chatResp.Body).Decode(&gen); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gen.Message.Role != llm.RoleAssistant || gen.Message.Content == "" {
		t.Fatalf("unexpected generation: %+v", gen)
	}
}

func TestChatEmptyMessages(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"messages": []interface{}{}})
	resp, err := http.Post(ts.URL+"/v1/llm-requests/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
