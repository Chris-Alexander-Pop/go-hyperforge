package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/services/encryption/server"
)

func TestHealthEncryptDecrypt(t *testing.T) {
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

	encBody, _ := json.Marshal(map[string]string{"plaintext": "secret-value"})
	encResp, err := http.Post(ts.URL+"/v1/encryption/encrypt", "application/json", bytes.NewReader(encBody))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	defer encResp.Body.Close()
	if encResp.StatusCode != http.StatusOK {
		t.Fatalf("encrypt status=%d", encResp.StatusCode)
	}
	var encOut struct {
		CiphertextBase64 string `json:"ciphertext_base64"`
	}
	if err := json.NewDecoder(encResp.Body).Decode(&encOut); err != nil {
		t.Fatalf("decode encrypt: %v", err)
	}
	if encOut.CiphertextBase64 == "" {
		t.Fatal("expected ciphertext")
	}

	decBody, _ := json.Marshal(map[string]string{"ciphertext_base64": encOut.CiphertextBase64})
	decResp, err := http.Post(ts.URL+"/v1/encryption/decrypt", "application/json", bytes.NewReader(decBody))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	defer decResp.Body.Close()
	if decResp.StatusCode != http.StatusOK {
		t.Fatalf("decrypt status=%d", decResp.StatusCode)
	}
	var decOut struct {
		Plaintext string `json:"plaintext"`
	}
	if err := json.NewDecoder(decResp.Body).Decode(&decOut); err != nil {
		t.Fatalf("decode decrypt: %v", err)
	}
	if decOut.Plaintext != "secret-value" {
		t.Fatalf("plaintext=%q", decOut.Plaintext)
	}
}

func TestEncryptMissingPlaintext(t *testing.T) {
	srv := server.New(server.Config{Port: "0"})
	ts := httptest.NewServer(srv.Echo())
	t.Cleanup(ts.Close)

	body, _ := json.Marshal(map[string]string{})
	resp, err := http.Post(ts.URL+"/v1/encryption/encrypt", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
