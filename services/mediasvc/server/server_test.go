package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/mediasvc/server"
)

func TestCreateAndGet(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"filename": "a.png", "content_type": "image/png", "data": "abc"})
	cr, err := http.Post(ts.URL+"/v1/media", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var asset map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&asset)
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", cr.StatusCode)
	}
	id, _ := asset["id"].(string)
	if asset["blob_ref"] == nil || asset["blob_ref"] == "" {
		t.Fatalf("expected blob_ref, got %v", asset)
	}

	gr, err := http.Get(ts.URL + "/v1/media/" + id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", gr.StatusCode)
	}
}

func TestCreateMissingFilename(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"content_type": "image/png"})
	cr, err := http.Post(ts.URL+"/v1/media", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", cr.StatusCode)
	}
}
