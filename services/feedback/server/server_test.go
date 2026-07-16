package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/feedback/server"
)

func TestSubmitAndList(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "target": "app", "rating": 5, "comment": "great"})
	sr, err := http.Post(ts.URL+"/v1/feedback", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	sr.Body.Close()
	if sr.StatusCode != http.StatusCreated {
		t.Fatalf("submit=%d", sr.StatusCode)
	}

	lr, err := http.Get(ts.URL + "/v1/feedback?target=app")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	defer lr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(lr.Body).Decode(&out)
	items, _ := out["feedback"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1, got %v", out)
	}
}

func TestSubmitBadRating(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]interface{}{"rating": 9, "comment": "x"})
	sr, err := http.Post(ts.URL+"/v1/feedback", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	defer sr.Body.Close()
	if sr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", sr.StatusCode)
	}
}
