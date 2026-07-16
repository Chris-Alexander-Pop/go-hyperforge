package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	emailmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/email/adapters/memory"
	pushmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/push/adapters/memory"
	smsmemory "github.com/chris-alexander-pop/go-hyperforge/pkg/communication/sms/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/services/notification/server"
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

func TestSendEmailChannel(t *testing.T) {
	emailMem := emailmemory.New()
	smsMem := smsmemory.New()
	pushMem := pushmemory.New()
	srv := server.NewWithSenders(server.Config{Port: "0"}, emailMem, smsMem, pushMem)
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"channel": "email",
		"to":      []string{"user@example.com"},
		"subject": "Notify",
		"text":    "Hello",
	})
	res, err := http.Post(ts.URL+"/v1/notifications/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("send status=%d", res.StatusCode)
	}

	msgs := emailMem.(*emailmemory.Sender).SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("email sent=%d", len(msgs))
	}
	if len(smsMem.(*smsmemory.Sender).SentMessages()) != 0 {
		t.Fatal("sms should be empty")
	}
	if len(pushMem.(*pushmemory.Sender).SentMessages()) != 0 {
		t.Fatal("push should be empty")
	}
}

func TestSendSMSAndPushChannels(t *testing.T) {
	emailMem := emailmemory.New()
	smsMem := smsmemory.New()
	pushMem := pushmemory.New()
	srv := server.NewWithSenders(server.Config{Port: "0"}, emailMem, smsMem, pushMem)
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	smsBody, _ := json.Marshal(map[string]string{
		"channel":  "sms",
		"to_phone": "+15550001111",
		"body":     "ping",
	})
	smsResp, err := http.Post(ts.URL+"/v1/notifications/send", "application/json", bytes.NewReader(smsBody))
	if err != nil {
		t.Fatalf("sms: %v", err)
	}
	smsResp.Body.Close()
	if smsResp.StatusCode != http.StatusAccepted {
		t.Fatalf("sms status=%d", smsResp.StatusCode)
	}

	pushBody, _ := json.Marshal(map[string]interface{}{
		"channel": "push",
		"tokens":  []string{"tok"},
		"title":   "Hi",
		"body":    "There",
	})
	pushResp, err := http.Post(ts.URL+"/v1/notifications/send", "application/json", bytes.NewReader(pushBody))
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	pushResp.Body.Close()
	if pushResp.StatusCode != http.StatusAccepted {
		t.Fatalf("push status=%d", pushResp.StatusCode)
	}

	if len(smsMem.(*smsmemory.Sender).SentMessages()) != 1 {
		t.Fatal("expected one sms")
	}
	if len(pushMem.(*pushmemory.Sender).SentMessages()) != 1 {
		t.Fatal("expected one push")
	}
}

func TestSendInvalidChannel(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"channel": "carrier-pigeon"})
	res, err := http.Post(ts.URL+"/v1/notifications/send", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
