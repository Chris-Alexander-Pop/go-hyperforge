package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/alerting/server"
)

func TestRuleFireListAck(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	rb, _ := json.Marshal(map[string]string{"name": "high-cpu", "query": "cpu > 90", "severity": "critical"})
	rr, err := http.Post(ts.URL+"/v1/alerts/rules", "application/json", bytes.NewReader(rb))
	if err != nil {
		t.Fatalf("rule: %v", err)
	}
	defer rr.Body.Close()
	var rule map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&rule)
	ruleID, _ := rule["id"].(string)

	fb, _ := json.Marshal(map[string]string{"rule_id": ruleID, "message": "cpu high"})
	fr, err := http.Post(ts.URL+"/v1/alerts/fire", "application/json", bytes.NewReader(fb))
	if err != nil {
		t.Fatalf("fire: %v", err)
	}
	defer fr.Body.Close()
	var alert map[string]interface{}
	json.NewDecoder(fr.Body).Decode(&alert)
	if fr.StatusCode != http.StatusCreated {
		t.Fatalf("fire status=%d", fr.StatusCode)
	}
	alertID, _ := alert["id"].(string)

	lr, _ := http.Get(ts.URL + "/v1/alerts")
	lr.Body.Close()
	if lr.StatusCode != http.StatusOK {
		t.Fatalf("list=%d", lr.StatusCode)
	}

	ar, err := http.Post(ts.URL+"/v1/alerts/"+alertID+"/ack", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	defer ar.Body.Close()
	if ar.StatusCode != http.StatusOK {
		t.Fatalf("ack status=%d", ar.StatusCode)
	}
}

func TestFireUnknownRule(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	fb, _ := json.Marshal(map[string]string{"rule_id": "missing", "message": "x"})
	fr, err := http.Post(ts.URL+"/v1/alerts/fire", "application/json", bytes.NewReader(fb))
	if err != nil {
		t.Fatalf("fire: %v", err)
	}
	defer fr.Body.Close()
	if fr.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", fr.StatusCode)
	}
}
