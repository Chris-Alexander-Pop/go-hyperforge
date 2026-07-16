package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/searchsvc/server"
)

func TestHealthIndexAndQuery(t *testing.T) {
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

	docBody, _ := json.Marshal(map[string]interface{}{
		"documents": []map[string]interface{}{
			{"id": "1", "document": map[string]interface{}{"title": "hello world", "body": "search me"}},
			{"id": "2", "document": map[string]interface{}{"title": "other", "body": "nothing"}},
		},
	})
	idxResp, err := http.Post(ts.URL+"/v1/search/indexes/products/documents", "application/json", bytes.NewReader(docBody))
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	defer idxResp.Body.Close()
	if idxResp.StatusCode != http.StatusCreated {
		t.Fatalf("index status=%d", idxResp.StatusCode)
	}

	qBody, _ := json.Marshal(map[string]string{"index": "products", "query": "hello"})
	qResp, err := http.Post(ts.URL+"/v1/search/query", "application/json", bytes.NewReader(qBody))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer qResp.Body.Close()
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("query status=%d", qResp.StatusCode)
	}
}

func TestQueryMissingIndex(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"query": "hello"})
	resp, err := http.Post(ts.URL+"/v1/search/query", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
