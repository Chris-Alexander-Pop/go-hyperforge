package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/inventory/server"
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

func TestUpsertReserveRelease(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	upsertBody, _ := json.Marshal(map[string]interface{}{"sku": "SKU-1", "quantity": 10})
	upsertResp, err := http.Post(ts.URL+"/v1/inventory/skus", "application/json", bytes.NewReader(upsertBody))
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	defer upsertResp.Body.Close()
	if upsertResp.StatusCode != http.StatusOK {
		t.Fatalf("upsert status=%d", upsertResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/inventory/skus/SKU-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", getResp.StatusCode)
	}

	reserveBody, _ := json.Marshal(map[string]int64{"qty": 3})
	reserveResp, err := http.Post(ts.URL+"/v1/inventory/skus/SKU-1/reserve", "application/json", bytes.NewReader(reserveBody))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	defer reserveResp.Body.Close()
	if reserveResp.StatusCode != http.StatusOK {
		t.Fatalf("reserve status=%d", reserveResp.StatusCode)
	}
	var reserved map[string]interface{}
	if err := json.NewDecoder(reserveResp.Body).Decode(&reserved); err != nil {
		t.Fatalf("decode reserve: %v", err)
	}
	if reserved["reserved"].(float64) != 3 {
		t.Fatalf("reserved=%v", reserved["reserved"])
	}
	if reserved["available"].(float64) != 7 {
		t.Fatalf("available=%v", reserved["available"])
	}

	releaseBody, _ := json.Marshal(map[string]int64{"qty": 1})
	releaseResp, err := http.Post(ts.URL+"/v1/inventory/skus/SKU-1/release", "application/json", bytes.NewReader(releaseBody))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	defer releaseResp.Body.Close()
	if releaseResp.StatusCode != http.StatusOK {
		t.Fatalf("release status=%d", releaseResp.StatusCode)
	}
}

func TestReserveInsufficient(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	upsertBody, _ := json.Marshal(map[string]interface{}{"sku": "SKU-1", "quantity": 2})
	upsertResp, _ := http.Post(ts.URL+"/v1/inventory/skus", "application/json", bytes.NewReader(upsertBody))
	upsertResp.Body.Close()

	reserveBody, _ := json.Marshal(map[string]int64{"qty": 5})
	res, err := http.Post(ts.URL+"/v1/inventory/skus/SKU-1/reserve", "application/json", bytes.NewReader(reserveBody))
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
