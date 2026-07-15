package avalara_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/tax/adapters/avalara"
)

func TestCalculateTaxSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !stringsHasPrefix(r.URL.Path, "/api/v2/transactions/create") {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "acct" || pass != "key" {
			t.Errorf("basic auth user=%q ok=%v", user, ok)
		}
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"totalTax":     7.25,
			"totalTaxable": 100.0,
			"totalAmount":  100.0,
			"currencyCode": "USD",
			"summary": []map[string]interface{}{
				{"country": "US", "region": "NY", "jurisType": "State", "rate": 0.04, "tax": 4.0},
				{"country": "US", "region": "NY", "jurisType": "City", "rate": 0.0325, "tax": 3.25},
			},
		})
	}))
	defer srv.Close()

	calc, err := avalara.New(avalara.Config{
		AccountID:        "acct",
		LicenseKey:       "key",
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := calc.CalculateTax(context.Background(), commerce.NewMoney(10000, "USD"), tax.Location{
		Country:    "US",
		State:      "NY",
		PostalCode: "10001",
		City:       "New York",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalTax.Amount != 725 {
		t.Fatalf("TotalTax=%d want 725", res.TotalTax.Amount)
	}
	if res.Breakdown["state"].Amount != 400 {
		t.Fatalf("state=%d", res.Breakdown["state"].Amount)
	}
	if res.Breakdown["city"].Amount != 325 {
		t.Fatalf("city=%d", res.Breakdown["city"].Amount)
	}
}

func TestCalculateTaxInvalidConfig(t *testing.T) {
	t.Parallel()
	_, err := avalara.New(avalara.Config{AccountID: "only"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCalculateTaxBadRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	calc, err := avalara.New(avalara.Config{
		AccountID:        "acct",
		LicenseKey:       "key",
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = calc.CalculateTax(context.Background(), commerce.NewMoney(100, "USD"), tax.Location{Country: "US"})
	if err != tax.ErrUnsupportedLocation {
		t.Fatalf("err=%v want ErrUnsupportedLocation", err)
	}
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
