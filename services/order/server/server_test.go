package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/order/server"
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

func TestCreateGetListCancel(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"customer_id": "cust_1",
		"currency":    "USD",
		"items": []map[string]interface{}{
			{"sku": "SKU-1", "qty": 2, "amount_minor": 500, "currency": "USD"},
		},
	})
	createResp, err := http.Post(ts.URL+"/v1/orders", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}

	var created map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected order id")
	}
	if created["status"] != "pending" {
		t.Fatalf("status=%v", created["status"])
	}
	if created["total_minor"].(float64) != 1000 {
		t.Fatalf("total_minor=%v want 1000", created["total_minor"])
	}

	getResp, err := http.Get(ts.URL + "/v1/orders/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	listResp, err := http.Get(ts.URL + "/v1/orders")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list status=%d", listResp.StatusCode)
	}

	cancelResp, err := http.Post(ts.URL+"/v1/orders/"+id+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("cancel status=%d", cancelResp.StatusCode)
	}
	var cancelled map[string]interface{}
	if err := json.NewDecoder(cancelResp.Body).Decode(&cancelled); err != nil {
		t.Fatalf("decode cancel: %v", err)
	}
	if cancelled["status"] != "cancelled" {
		t.Fatalf("status=%v", cancelled["status"])
	}
}

func TestCreateMissingCustomer(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{
		"currency": "USD",
		"items":    []map[string]interface{}{{"sku": "SKU-1", "qty": 1, "amount_minor": 100}},
	})
	res, err := http.Post(ts.URL+"/v1/orders", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}

func TestGetNotFound(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/v1/orders/missing")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}
}
