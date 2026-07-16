package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/taxcalculator/server"
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

func TestCalculateTax(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"amount_minor": 10000,
		"currency":     "USD",
		"location": map[string]string{
			"country": "US",
			"state":   "NY",
		},
	})
	res, err := http.Post(ts.URL+"/v1/taxes/calculate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("calculate status=%d", res.StatusCode)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	total, ok := got["total_tax"].(map[string]interface{})
	if !ok {
		t.Fatalf("total_tax missing: %v", got)
	}
	if total["amount_minor"].(float64) <= 0 {
		t.Fatalf("expected positive tax, got %v", total["amount_minor"])
	}
}

func TestCalculateInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/taxes/calculate", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("calculate: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
