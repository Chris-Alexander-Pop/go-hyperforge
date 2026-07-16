package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/recommendation/server"
)

func TestTrackAndRecommend(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	for _, pair := range [][2]string{{"u1", "a"}, {"u1", "b"}, {"u2", "a"}, {"u2", "c"}} {
		body, _ := json.Marshal(map[string]string{"user_id": pair[0], "item_id": pair[1]})
		tr, err := http.Post(ts.URL+"/v1/recommendations/interactions", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("track: %v", err)
		}
		tr.Body.Close()
		if tr.StatusCode != http.StatusCreated {
			t.Fatalf("track=%d", tr.StatusCode)
		}
	}

	rb, _ := json.Marshal(map[string]interface{}{"user_id": "u1", "limit": 3})
	rr, err := http.Post(ts.URL+"/v1/recommendations", "application/json", bytes.NewReader(rb))
	if err != nil {
		t.Fatalf("recommend: %v", err)
	}
	defer rr.Body.Close()
	var out map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&out)
	if rr.StatusCode != http.StatusOK {
		t.Fatalf("recommend=%d out=%v", rr.StatusCode, out)
	}
	recs, _ := out["recommendations"].([]interface{})
	if len(recs) == 0 {
		t.Fatalf("expected recommendations, got %v", out)
	}
}

func TestRecommendMissingUser(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	rb, _ := json.Marshal(map[string]interface{}{"limit": 3})
	rr, err := http.Post(ts.URL+"/v1/recommendations", "application/json", bytes.NewReader(rb))
	if err != nil {
		t.Fatalf("recommend: %v", err)
	}
	defer rr.Body.Close()
	if rr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.StatusCode)
	}
}
