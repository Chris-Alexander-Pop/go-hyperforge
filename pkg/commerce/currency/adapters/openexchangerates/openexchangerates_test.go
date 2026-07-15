package openexchangerates_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	cachememory "github.com/chris-alexander-pop/go-hyperforge/pkg/cache/adapters/memory"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/currency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/commerce/currency/adapters/openexchangerates"
)

func TestFetchRatesOpenExchangeShape(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/latest.json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("app_id") != "app123" {
			t.Errorf("app_id=%s", r.URL.Query().Get("app_id"))
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"base":      "USD",
			"timestamp": time.Now().Unix(),
			"rates": map[string]float64{
				"EUR": 0.92,
				"GBP": 0.79,
			},
		})
	}))
	defer srv.Close()

	p, err := openexchangerates.New(openexchangerates.Config{
		AppID:            "app123",
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	rates, err := p.FetchRates(context.Background(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	if rates["EUR"] != 0.92 || rates["USD"] != 1.0 {
		t.Fatalf("rates=%v", rates)
	}

	res, err := p.Convert(context.Background(), 100, "USD", "EUR")
	if err != nil {
		t.Fatal(err)
	}
	if res.ToAmount != 92.0 {
		t.Fatalf("ToAmount=%v", res.ToAmount)
	}
}

func TestFrankfurterShapeAndCache(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/latest" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"base": "USD",
			"date": "2024-06-01",
			"rates": map[string]float64{
				"EUR": 0.91,
				"JPY": 157.0,
			},
		})
	}))
	defer srv.Close()

	c := cachememory.New()
	defer c.Close()

	p, err := openexchangerates.New(openexchangerates.Config{
		BaseURL:          srv.URL, // not openexchangerates.org → frankfurter-style URL
		Cache:            c,
		CacheTTL:         time.Hour,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := p.FetchRates(context.Background(), "USD"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.FetchRates(context.Background(), "USD"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 HTTP hit (cache), got %d", hits.Load())
	}

	rate, err := p.GetRate(context.Background(), "EUR", "JPY")
	if err != nil {
		t.Fatal(err)
	}
	want := 157.0 / 0.91
	if rate < want-0.001 || rate > want+0.001 {
		t.Fatalf("rate=%v want ~%v", rate, want)
	}
}

func TestRequiresAppIDForOER(t *testing.T) {
	t.Parallel()
	_, err := openexchangerates.New(openexchangerates.Config{
		BaseURL: "https://openexchangerates.org/api",
	})
	if err == nil {
		t.Fatal("expected app id required")
	}
}

func TestLiveRatesUnavailable(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	p, err := openexchangerates.New(openexchangerates.Config{
		BaseURL:          srv.URL,
		RetryMaxAttempts: 1,
		HTTPClient:       srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.FetchRates(context.Background(), "USD")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	t.Parallel()
	var _ currency.Converter = (*openexchangerates.Provider)(nil)
	var _ currency.LiveRateProvider = (*openexchangerates.Provider)(nil)
}
