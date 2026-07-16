package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/payment/server"
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

func TestChargeThenGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0", PaymentProvider: "memory"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"amount_minor":    2500,
		"currency":        "USD",
		"source_id":       "tok_test",
		"description":     "test charge",
		"idempotency_key": "idem-1",
	})
	chargeResp, err := http.Post(ts.URL+"/v1/payments/charge", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	defer chargeResp.Body.Close()
	if chargeResp.StatusCode != http.StatusOK {
		t.Fatalf("charge status=%d", chargeResp.StatusCode)
	}

	var charged map[string]interface{}
	if err := json.NewDecoder(chargeResp.Body).Decode(&charged); err != nil {
		t.Fatalf("decode charge: %v", err)
	}
	id, _ := charged["id"].(string)
	if id == "" {
		t.Fatal("expected transaction id")
	}
	if charged["amount_minor"].(float64) != 2500 {
		t.Fatalf("amount_minor=%v", charged["amount_minor"])
	}
	if charged["status"] != "succeeded" {
		t.Fatalf("status=%v", charged["status"])
	}

	getResp, err := http.Get(ts.URL + "/v1/payments/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got["id"] != id {
		t.Fatalf("got id=%v want %s", got["id"], id)
	}
}

func TestChargeInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/payments/charge", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}

func TestChargeMissingCurrency(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"amount_minor": 100,
		"source_id":    "tok_test",
	})
	res, err := http.Post(ts.URL+"/v1/payments/charge", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
