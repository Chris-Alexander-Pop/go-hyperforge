package server_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/blobstorage/server"
)

func TestHealthUploadDownloadDelete(t *testing.T) {
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

	payload := map[string]string{
		"key":            "docs/hello.txt",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("hello blob")),
	}
	body, _ := json.Marshal(payload)
	upResp, err := http.Post(ts.URL+"/v1/blobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer upResp.Body.Close()
	if upResp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status=%d", upResp.StatusCode)
	}

	getResp, err := http.Get(ts.URL + "/v1/blobs/docs/hello.txt")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("download status=%d", getResp.StatusCode)
	}
	var got struct {
		ContentBase64 string `json:"content_base64"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	raw, _ := base64.StdEncoding.DecodeString(got.ContentBase64)
	if string(raw) != "hello blob" {
		t.Fatalf("content=%q", raw)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/blobs/docs/hello.txt", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	delResp.Body.Close()
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", delResp.StatusCode)
	}
}

func TestUploadMissingKey(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{"content_base64": base64.StdEncoding.EncodeToString([]byte("x"))})
	resp, err := http.Post(ts.URL+"/v1/blobs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
