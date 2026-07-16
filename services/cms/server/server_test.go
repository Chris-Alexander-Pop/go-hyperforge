package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/cms/server"
)

func TestCreatePublishGetBySlug(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	res, _ := http.Get(ts.URL + "/healthz")
	res.Body.Close()

	body, _ := json.Marshal(map[string]string{"slug": "about", "title": "About", "body": "hi"})
	cr, err := http.Post(ts.URL+"/v1/pages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	var page map[string]interface{}
	json.NewDecoder(cr.Body).Decode(&page)
	id, _ := page["id"].(string)

	pr, err := http.Post(ts.URL+"/v1/pages/"+id+"/publish", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	pr.Body.Close()
	if pr.StatusCode != http.StatusOK {
		t.Fatalf("publish=%d", pr.StatusCode)
	}

	gr, err := http.Get(ts.URL + "/v1/pages/by-slug/about")
	if err != nil {
		t.Fatalf("by-slug: %v", err)
	}
	defer gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("by-slug=%d", gr.StatusCode)
	}
}

func TestCreateMissingSlug(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]string{"title": "About"})
	cr, err := http.Post(ts.URL+"/v1/pages", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer cr.Body.Close()
	if cr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", cr.StatusCode)
	}
}
