package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/webhookmanager/server"
)

func TestRegisterAndDeliver(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	rb, _ := json.Marshal(map[string]interface{}{"url": "https://example.com/hook", "events": []string{"order.created"}})
	rr, err := http.Post(ts.URL+"/v1/webhooks", "application/json", bytes.NewReader(rb))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	defer rr.Body.Close()
	var ep map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&ep)
	id, _ := ep["id"].(string)

	db, _ := json.Marshal(map[string]string{"event": "order.created", "payload": "{}"})
	dr, err := http.Post(ts.URL+"/v1/webhooks/"+id+"/deliveries", "application/json", bytes.NewReader(db))
	if err != nil {
		t.Fatalf("delivery: %v", err)
	}
	dr.Body.Close()
	if dr.StatusCode != http.StatusCreated {
		t.Fatalf("delivery status=%d", dr.StatusCode)
	}
}

func TestDeliveryUnknownEndpoint(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	db, _ := json.Marshal(map[string]string{"event": "x"})
	dr, err := http.Post(ts.URL+"/v1/webhooks/missing/deliveries", "application/json", bytes.NewReader(db))
	if err != nil {
		t.Fatalf("delivery: %v", err)
	}
	defer dr.Body.Close()
	if dr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", dr.StatusCode)
	}
}
