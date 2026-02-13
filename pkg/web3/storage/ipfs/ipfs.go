// Package ipfs provides an IPFS client for decentralized storage.
//
// Supports adding, getting, and pinning content on IPFS.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/system-design-library/pkg/web3/storage/ipfs"
//
//	client, err := ipfs.New(ipfs.Config{APIURL: "http://localhost:5001"})
//	cid, err := client.Add(ctx, data)
package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	pkgerrors "github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Config holds IPFS client configuration.
type Config struct {
	// APIURL is the IPFS HTTP API endpoint
	APIURL string

	// GatewayURL is the IPFS gateway for retrieving content
	GatewayURL string
}

// Client provides IPFS access.
type Client struct {
	apiURL     string
	gatewayURL string
	httpClient *http.Client
}

// New creates a new IPFS client.
func New(cfg Config) (*Client, error) {
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:5001"
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = "https://ipfs.io"
	}

	return &Client{
		apiURL:     cfg.APIURL,
		gatewayURL: cfg.GatewayURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

// Add uploads content to IPFS and returns the CID.
func (c *Client) Add(ctx context.Context, data []byte) (string, error) {
	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "data")
	if err != nil {
		return "", pkgerrors.Internal("failed to create form", err)
	}

	if _, err := part.Write(data); err != nil {
		return "", pkgerrors.Internal("failed to write data", err)
	}
	writer.Close()

	url := fmt.Sprintf("%s/api/v0/add", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return "", pkgerrors.Internal("failed to create request", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", pkgerrors.Internal("failed to upload to IPFS", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", pkgerrors.Internal("IPFS add failed: "+string(body), nil)
	}

	var result struct {
		Hash string `json:"Hash"`
		Name string `json:"Name"`
		Size string `json:"Size"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", pkgerrors.Internal("failed to parse response", err)
	}

	return result.Hash, nil
}

// AddJSON uploads JSON data to IPFS.
func (c *Client) AddJSON(ctx context.Context, data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", pkgerrors.Internal("failed to marshal JSON", err)
	}
	return c.Add(ctx, jsonData)
}

// Get retrieves content from IPFS by CID.
func (c *Client) Get(ctx context.Context, cid string) ([]byte, error) {
	url := fmt.Sprintf("%s/api/v0/cat?arg=%s", c.apiURL, cid)
	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get from IPFS", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, pkgerrors.NotFound("content not found", nil)
	}

	return io.ReadAll(resp.Body)
}

// GetJSON retrieves and parses JSON from IPFS.
func (c *Client) GetJSON(ctx context.Context, cid string, result interface{}) error {
	data, err := c.Get(ctx, cid)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, result)
}

// GetURL returns the gateway URL for a CID.
func (c *Client) GetURL(cid string) string {
	return fmt.Sprintf("%s/ipfs/%s", c.gatewayURL, cid)
}

// Pin pins content to prevent garbage collection.
func (c *Client) Pin(ctx context.Context, cid string) error {
	url := fmt.Sprintf("%s/api/v0/pin/add?arg=%s", c.apiURL, cid)
	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	if err != nil {
		return pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pkgerrors.Internal("failed to pin", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return pkgerrors.Internal("pin failed: "+string(body), nil)
	}

	return nil
}

// Unpin removes a pin from content.
func (c *Client) Unpin(ctx context.Context, cid string) error {
	url := fmt.Sprintf("%s/api/v0/pin/rm?arg=%s", c.apiURL, cid)
	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	if err != nil {
		return pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pkgerrors.Internal("failed to unpin", err)
	}
	defer resp.Body.Close()

	return nil
}

// ListPins returns all pinned CIDs.
func (c *Client) ListPins(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/v0/pin/ls", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to list pins", err)
	}
	defer resp.Body.Close()

	var result struct {
		Keys map[string]struct {
			Type string `json:"Type"`
		} `json:"Keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	pins := make([]string, 0, len(result.Keys))
	for cid := range result.Keys {
		pins = append(pins, cid)
	}

	return pins, nil
}

// ID returns the node's peer ID and addresses.
func (c *Client) ID(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v0/id", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)
	if err != nil {
		return nil, pkgerrors.Internal("failed to create request", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.Internal("failed to get ID", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, pkgerrors.Internal("failed to parse response", err)
	}

	return result, nil
}
