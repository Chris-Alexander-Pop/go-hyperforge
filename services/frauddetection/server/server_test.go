package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/security/fraud"
	"github.com/chris-alexander-pop/go-hyperforge/services/frauddetection/server"
)

func TestHealthAndScore(t *testing.T) {
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

	body, _ := json.Marshal(fraud.UserEvent{
		UserID:    "u1",
		IPAddress: "8.8.8.8",
		Action:    "purchase",
		Amount:    42,
		Currency:  "USD",
	})
	scoreResp, err := http.Post(ts.URL+"/v1/fraud/score", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	defer scoreResp.Body.Close()
	if scoreResp.StatusCode != http.StatusOK {
		t.Fatalf("score status=%d", scoreResp.StatusCode)
	}

	var eval fraud.Evaluation
	if err := json.NewDecoder(scoreResp.Body).Decode(&eval); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if eval.CheckID == "" || eval.Action == "" {
		t.Fatalf("unexpected evaluation: %+v", eval)
	}
}

func TestScoreEmptyEvent(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]interface{}{})
	resp, err := http.Post(ts.URL+"/v1/fraud/score", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
