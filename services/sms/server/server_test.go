package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	smsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/services/sms/server"
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

func TestSendSMS(t *testing.T) {
	mem := smsmemory.New()
	srv := server.NewWithSender(server.Config{Port: "0"}, mem)
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{
		"to":   "+15551234567",
		"body": "hello",
	})
	res, err := http.Post(ts.URL+"/v1/sms/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("send status=%d", res.StatusCode)
	}

	concrete, ok := mem.(*smsmemory.Sender)
	if !ok {
		t.Fatal("expected *smsmemory.Sender")
	}
	msgs := concrete.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent=%d", len(msgs))
	}
	if msgs[0].To != "+15551234567" {
		t.Fatalf("to=%q", msgs[0].To)
	}
}

func TestSendInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/sms/send", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
