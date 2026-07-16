package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pushmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/services/pushnotification/server"
)

func TestHealthz(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", res.StatusCode)
	}
}

func TestSendPush(t *testing.T) {
	mem := pushmemory.New()
	srv := server.NewWithSender(server.Config{Port: "0"}, mem)
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"tokens": []string{"device-token-1"},
		"title":  "Alert",
		"body":   "Something happened",
	})
	res, err := http.Post(ts.URL+"/v1/pushes/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("send status=%d", res.StatusCode)
	}

	concrete, ok := mem.(*pushmemory.Sender)
	if !ok {
		t.Fatal("expected *pushmemory.Sender")
	}
	msgs := concrete.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent=%d", len(msgs))
	}
	if msgs[0].Title != "Alert" {
		t.Fatalf("title=%q", msgs[0].Title)
	}
}

func TestSendInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/pushes/send", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
