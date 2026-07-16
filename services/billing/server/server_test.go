package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/billing/server"
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

func TestPlansSubscribeCancelAndInvoices(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	plansResp, err := http.Get(ts.URL + "/v1/bills/plans")
	if err != nil {
		t.Fatalf("plans: %v", err)
	}
	defer plansResp.Body.Close()
	if plansResp.StatusCode != http.StatusOK {
		t.Fatalf("plans status=%d", plansResp.StatusCode)
	}
	var plans []map[string]interface{}
	if err := json.NewDecoder(plansResp.Body).Decode(&plans); err != nil {
		t.Fatalf("decode plans: %v", err)
	}
	if len(plans) == 0 {
		t.Fatal("expected built-in plans")
	}

	body, _ := json.Marshal(map[string]string{
		"customer_id": "cust_1",
		"plan_id":     "basic_monthly",
	})
	subResp, err := http.Post(ts.URL+"/v1/bills/subscriptions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer subResp.Body.Close()
	if subResp.StatusCode != http.StatusCreated {
		t.Fatalf("subscribe status=%d", subResp.StatusCode)
	}
	var sub map[string]interface{}
	if err := json.NewDecoder(subResp.Body).Decode(&sub); err != nil {
		t.Fatalf("decode sub: %v", err)
	}
	subID, _ := sub["id"].(string)
	if subID == "" {
		t.Fatal("expected subscription id")
	}

	getResp, err := http.Get(ts.URL + "/v1/bills/subscriptions/" + subID)
	if err != nil {
		t.Fatalf("get sub: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("get sub status=%d", getResp.StatusCode)
	}

	cancelResp, err := http.Post(ts.URL+"/v1/bills/subscriptions/"+subID+"/cancel", "application/json", nil)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	defer cancelResp.Body.Close()
	if cancelResp.StatusCode != http.StatusOK {
		t.Fatalf("cancel status=%d", cancelResp.StatusCode)
	}

	invBody, _ := json.Marshal(map[string]interface{}{
		"customer_id":  "cust_1",
		"amount_minor": 1500,
		"currency":     "USD",
	})
	invResp, err := http.Post(ts.URL+"/v1/bills/invoices", "application/json", bytes.NewReader(invBody))
	if err != nil {
		t.Fatalf("invoice: %v", err)
	}
	defer invResp.Body.Close()
	if invResp.StatusCode != http.StatusCreated {
		t.Fatalf("invoice status=%d", invResp.StatusCode)
	}

	listResp, err := http.Get(ts.URL + "/v1/bills/invoices?customer_id=cust_1")
	if err != nil {
		t.Fatalf("list invoices: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("list invoices status=%d", listResp.StatusCode)
	}
	var invoices []map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&invoices); err != nil {
		t.Fatalf("decode invoices: %v", err)
	}
	if len(invoices) == 0 {
		t.Fatal("expected at least one invoice")
	}
}

func TestCreateSubscriptionInvalidBody(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Post(ts.URL+"/v1/bills/subscriptions", "application/json", bytes.NewReader([]byte(`{`)))
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 400 || res.StatusCode >= 500 {
		t.Fatalf("expected 4xx, got %d", res.StatusCode)
	}
}
