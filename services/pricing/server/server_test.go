package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/pricing/server"
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

func TestCreateQuoteList(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"sku": "SKU-1", "amount_minor": 400, "currency": "USD", "name": "standard",
	})
	createResp, err := http.Post(ts.URL+"/v1/prices", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}
	var created map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id, _ := created["id"].(string)

	getResp, err := http.Get(ts.URL + "/v1/prices/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	quoteBody, _ := json.Marshal(map[string]interface{}{"sku": "SKU-1", "qty": 3})
	quoteResp, err := http.Post(ts.URL+"/v1/prices/quote", "application/json", bytes.NewReader(quoteBody))
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	defer quoteResp.Body.Close()
	if quoteResp.StatusCode != http.StatusOK {
		t.Fatalf("quote status=%d", quoteResp.StatusCode)
	}
	var quote map[string]interface{}
	if err := json.NewDecoder(quoteResp.Body).Decode(&quote); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if quote["unit"].(float64) != 400 {
		t.Fatalf("unit=%v", quote["unit"])
	}
	if quote["total"].(float64) != 1200 {
		t.Fatalf("total=%v", quote["total"])
	}

	listResp, err := http.Get(ts.URL + "/v1/prices")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listResp.StatusCode)
	}
}

func TestQuoteMissingSKU(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"sku": "missing", "qty": 1})
	res, err := http.Post(ts.URL+"/v1/prices/quote", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("quote: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
