package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/chat/server"
)

func TestRoomsAndMessages(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	rb, _ := json.Marshal(map[string]string{"name": "general"})
	rr, err := http.Post(ts.URL+"/v1/chats/rooms", "application/json", bytes.NewReader(rb))
	if err != nil {
		t.Fatalf("room: %v", err)
	}
	defer rr.Body.Close()
	var room map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&room)
	id, _ := room["id"].(string)

	mb, _ := json.Marshal(map[string]string{"user_id": "u1", "text": "hello"})
	mr, err := http.Post(ts.URL+"/v1/chats/rooms/"+id+"/messages", "application/json", bytes.NewReader(mb))
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	mr.Body.Close()
	if mr.StatusCode != http.StatusCreated {
		t.Fatalf("message status=%d", mr.StatusCode)
	}

	lr, err := http.Get(ts.URL + "/v1/chats/rooms/" + id + "/messages")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer lr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(lr.Body).Decode(&out)
	msgs, _ := out["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %v", out)
	}
}

func TestMessageUnknownRoom(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	mb, _ := json.Marshal(map[string]string{"user_id": "u1", "text": "hello"})
	mr, err := http.Post(ts.URL+"/v1/chats/rooms/missing/messages", "application/json", bytes.NewReader(mb))
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	defer mr.Body.Close()
	if mr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", mr.StatusCode)
	}
}
