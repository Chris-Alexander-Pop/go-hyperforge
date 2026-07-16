package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/dataretention/server"
)

func TestCreateListEvaluate(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", res.StatusCode)
	}

	body, _ := json.Marshal(map[string]interface{}{"resource": "logs", "days": 30})
	createResp, err := http.Post(ts.URL+"/v1/retention/policies", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", createResp.StatusCode)
	}

	listResp, err := http.Get(ts.URL + "/v1/retention/policies")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer listResp.Body.Close()
	var listed struct {
		Policies []struct {
			Resource string `json:"resource"`
			Days     int    `json:"days"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed.Policies) != 1 || listed.Policies[0].Resource != "logs" {
		t.Fatalf("unexpected list: %+v", listed)
	}

	evalResp, err := http.Post(ts.URL+"/v1/retention/evaluate", "application/json", nil)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	defer evalResp.Body.Close()
	if evalResp.StatusCode != http.StatusOK {
		t.Fatalf("evaluate status=%d", evalResp.StatusCode)
	}
	var eval struct {
		PoliciesEvaluated int `json:"policies_evaluated"`
		ExpiredDeleted    int `json:"expired_deleted"`
	}
	if err := json.NewDecoder(evalResp.Body).Decode(&eval); err != nil {
		t.Fatalf("decode eval: %v", err)
	}
	if eval.PoliciesEvaluated != 1 || eval.ExpiredDeleted != 0 {
		t.Fatalf("unexpected eval: %+v", eval)
	}
}

func TestCreateInvalidDays(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{"resource": "logs", "days": 0})
	resp, err := http.Post(ts.URL+"/v1/retention/policies", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
