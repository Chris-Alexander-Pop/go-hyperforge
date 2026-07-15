// Package kubo implements web3.Store against a Kubo (go-ipfs) HTTP API.
//
// Prefer this adapter (or adapters/memory) over importing storage/ipfs directly;
// the ipfs package is a thin wrapper around this adapter.
package kubo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
)

// Ensure compile-time interface compliance.
var _ web3.Store = (*Store)(nil)

// Config holds Kubo / IPFS HTTP API configuration.
type Config struct {
	// APIURL is the IPFS HTTP API endpoint (default http://localhost:5001).
	APIURL string

	// GatewayURL is the IPFS gateway for retrieving content (default https://ipfs.io).
	GatewayURL string

	// HTTPClient is optional; defaults to a 60s timeout client.
	HTTPClient *http.Client
}

// Store implements web3.Store via the Kubo HTTP API.
type Store struct {
	apiURL     string
	gatewayURL string
	httpClient *http.Client
}

// New creates a Kubo-backed web3.Store.
func New(cfg Config) (*Store, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:5001"
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = "https://ipfs.io"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Store{
		apiURL:     cfg.APIURL,
		gatewayURL: cfg.GatewayURL,
		httpClient: client,
	}, nil
}

// WithHTTPClient overrides the HTTP client (tests).
func (s *Store) WithHTTPClient(c *http.Client) *Store {
	if c != nil {
		s.httpClient = c
	}
	return s
}

// Add uploads content and returns its CID.
func (s *Store) Add(ctx context.Context, data []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", web3.ErrStorageFailed("Add", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", web3.ErrStorageFailed("Add", err)
	}
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/api/v0/add", body)
	if err != nil {
		return "", web3.ErrStorageFailed("Add", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", web3.ErrStorageFailed("Add", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", web3.ErrStorageFailed("Add", errors.New(web3.CodeStorageFailed, string(b), nil))
	}
	var result struct {
		Hash string `json:"Hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", web3.ErrStorageFailed("Add", err)
	}
	return result.Hash, nil
}

// AddJSON marshals and uploads JSON data.
func (s *Store) AddJSON(ctx context.Context, data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", web3.ErrStorageFailed("AddJSON", err)
	}
	return s.Add(ctx, jsonData)
}

// Get retrieves content by CID.
func (s *Store) Get(ctx context.Context, cid string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/api/v0/cat?arg="+cid, http.NoBody)
	if err != nil {
		return nil, web3.ErrStorageFailed("Get", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, web3.ErrStorageFailed("Get", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, web3.ErrNotFound("content", nil)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, web3.ErrStorageFailed("Get", err)
	}
	return data, nil
}

// GetJSON retrieves and unmarshals JSON by CID.
func (s *Store) GetJSON(ctx context.Context, cid string, result interface{}) error {
	data, err := s.Get(ctx, cid)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, result); err != nil {
		return web3.ErrStorageFailed("GetJSON", err)
	}
	return nil
}

// GetURL returns a gateway-style URL for a CID.
func (s *Store) GetURL(cid string) string {
	return fmt.Sprintf("%s/ipfs/%s", s.gatewayURL, cid)
}

// Pin marks content to prevent garbage collection.
func (s *Store) Pin(ctx context.Context, cid string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/api/v0/pin/add?arg="+cid, http.NoBody)
	if err != nil {
		return web3.ErrStorageFailed("Pin", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return web3.ErrStorageFailed("Pin", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return web3.ErrStorageFailed("Pin", errors.New(web3.CodeStorageFailed, string(b), nil))
	}
	return nil
}

// Unpin removes a pin.
func (s *Store) Unpin(ctx context.Context, cid string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/api/v0/pin/rm?arg="+cid, http.NoBody)
	if err != nil {
		return web3.ErrStorageFailed("Unpin", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return web3.ErrStorageFailed("Unpin", err)
	}
	defer resp.Body.Close()
	return nil
}

// ListPins returns all pinned CIDs.
func (s *Store) ListPins(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL+"/api/v0/pin/ls", http.NoBody)
	if err != nil {
		return nil, web3.ErrStorageFailed("ListPins", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, web3.ErrStorageFailed("ListPins", err)
	}
	defer resp.Body.Close()
	var result struct {
		Keys map[string]struct {
			Type string `json:"Type"`
		} `json:"Keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, web3.ErrStorageFailed("ListPins", err)
	}
	pins := make([]string, 0, len(result.Keys))
	for cid := range result.Keys {
		pins = append(pins, cid)
	}
	return pins, nil
}
