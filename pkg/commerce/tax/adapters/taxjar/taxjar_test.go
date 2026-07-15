package taxjar_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax/adapters/taxjar"
)

func TestCalculateTaxSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v2/taxes" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("auth header = %q", got)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req["to_country"] != "US" || req["to_state"] != "CA" {
			t.Errorf("unexpected destination: %#v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"tax": map[string]interface{}{
				"order_total_amount": 100.0,
				"taxable_amount":     100.0,
				"amount_to_collect":  8.25,
				"rate":               0.0825,
				"jurisdictions": map[string]string{
					"country": "US",
					"state":   "CA",
					"city":    "LOS ANGELES",
				},
				"breakdown": map[string]interface{}{
					"tax_collectable":       8.25,
					"combined_tax_rate":     0.0825,
					"state_tax_collectable": 6.25,
					"state_tax_rate":        0.0625,
					"city_tax_collectable":  2.0,
				},
			},
		})
	}))
	defer srv.Close()

	calc, err := taxjar.New(taxjar.Config{
		APIToken:         "test-token",
		BaseURL:          srv.URL,
		FromCountry:      "US",
		FromState:        "CA",
		FromZip:          "92093",
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := calc.CalculateTax(context.Background(), commerce.NewMoney(10000, "USD"), tax.Location{
		Country:    "US",
		State:      "CA",
		PostalCode: "90002",
		City:       "Los Angeles",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalTax.Amount != 825 {
		t.Fatalf("TotalTax=%d want 825", res.TotalTax.Amount)
	}
	if res.Rate != 0.0825 {
		t.Fatalf("Rate=%v want 0.0825", res.Rate)
	}
	if res.Jurisdiction.State != "CA" {
		t.Fatalf("jurisdiction state=%s", res.Jurisdiction.State)
	}
	if res.Breakdown["state"].Amount != 625 {
		t.Fatalf("state breakdown=%d", res.Breakdown["state"].Amount)
	}
}

func TestCalculateTaxInvalidConfig(t *testing.T) {
	t.Parallel()
	_, err := taxjar.New(taxjar.Config{})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestCalculateTaxInvalidAmount(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API")
	}))
	defer srv.Close()

	calc, err := taxjar.New(taxjar.Config{
		APIToken:         "tok",
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = calc.CalculateTax(context.Background(), commerce.NewMoney(-1, "USD"), tax.Location{Country: "US"})
	if err != tax.ErrInvalidAmount {
		t.Fatalf("err=%v want ErrInvalidAmount", err)
	}
}

func TestCalculateTaxServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	calc, err := taxjar.New(taxjar.Config{
		APIToken:         "tok",
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = calc.CalculateTax(context.Background(), commerce.NewMoney(100, "USD"), tax.Location{Country: "US", State: "NY"})
	if err == nil {
		t.Fatal("expected error")
	}
}
