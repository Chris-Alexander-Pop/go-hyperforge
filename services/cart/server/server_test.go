package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/cart/server"
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

func TestCartFlow(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	createBody, _ := json.Marshal(map[string]string{"user_id": "user_1"})
	createResp, err := http.Post(ts.URL+"/v1/carts", "application/json", bytes.NewReader(createBody))
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
		t.Fatal("expected cart id")
	}

	itemBody, _ := json.Marshal(map[string]interface{}{
		"sku": "SKU-A", "qty": 2, "amount_minor": 250, "currency": "USD",
	})
	addResp, err := http.Post(ts.URL+"/v1/carts/"+id+"/items", "application/json", bytes.NewReader(itemBody))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	defer addResp.Body.Close()
	if addResp.StatusCode != http.StatusOK {
		t.Fatalf("add status=%d", addResp.StatusCode)
	}

	checkoutResp, err := http.Post(ts.URL+"/v1/carts/"+id+"/checkout", "application/json", nil)
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	defer checkoutResp.Body.Close()
	if checkoutResp.StatusCode != http.StatusOK {
		t.Fatalf("checkout status=%d", checkoutResp.StatusCode)
	}
	var checkout map[string]interface{}
	if err := json.NewDecoder(checkoutResp.Body).Decode(&checkout); err != nil {
		t.Fatalf("decode checkout: %v", err)
	}
	if checkout["total_minor"].(float64) != 500 {
		t.Fatalf("total_minor=%v want 500", checkout["total_minor"])
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/carts/"+id+"/items/SKU-A", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}
}

func TestCheckoutEmptyCart(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	createResp, err := http.Post(ts.URL+"/v1/carts", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	var created map[string]interface{}
	_ = json.NewDecoder(createResp.Body).Decode(&created)
	id, _ := created["id"].(string)

	res, err := http.Post(ts.URL+"/v1/carts/"+id+"/checkout", "application/json", nil)
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
