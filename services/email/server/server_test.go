package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	emailmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/services/email/server"
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

func TestSendEmail(t *testing.T) {
	mem := emailmemory.New()
	srv := server.NewWithSender(server.Config{Port: "0"}, mem)
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"to":      []string{"user@example.com"},
		"subject": "Hello",
		"text":    "Hi there",
	})
	res, err := http.Post(ts.URL+"/v1/emails/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("send status=%d", res.StatusCode)
	}

	concrete, ok := mem.(*emailmemory.Sender)
	if !ok {
		t.Fatal("expected *emailmemory.Sender")
	}
	msgs := concrete.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent=%d", len(msgs))
	}
	if msgs[0].Subject != "Hello" {
		t.Fatalf("subject=%q", msgs[0].Subject)
	}
}

func TestSendInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/emails/send", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
