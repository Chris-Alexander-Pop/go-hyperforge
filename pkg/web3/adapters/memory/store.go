package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/concurrency"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
)

// Ensure compile-time interface compliance.
var _ web3.Store = (*Store)(nil)

// StoreConfig configures the in-memory IPFS store.
type StoreConfig struct {
	// GatewayURL defaults to https://ipfs.io when empty.
	GatewayURL string
}

// Store is an in-memory IPFS-like content store for tests.
type Store struct {
	mu         *concurrency.SmartRWMutex
	content    map[string][]byte
	pins       map[string]struct{}
	gatewayURL string
}

// NewStore creates an in-memory IPFS store.
func NewStore(cfg StoreConfig) *Store {
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = "https://ipfs.io"
	}
	return &Store{
		mu:         concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "web3-memory-store"}),
		content:    make(map[string][]byte),
		pins:       make(map[string]struct{}),
		gatewayURL: cfg.GatewayURL,
	}
}

// Add stores data and returns a deterministic content id.
func (s *Store) Add(ctx context.Context, data []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cid := ContentID(data)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content[cid] = append([]byte(nil), data...)
	return cid, nil
}

// AddJSON marshals and stores JSON data.
func (s *Store) AddJSON(ctx context.Context, data interface{}) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", web3.ErrStorageFailed("marshal JSON", err)
	}
	return s.Add(ctx, jsonData)
}

// Get retrieves content by CID.
func (s *Store) Get(ctx context.Context, cid string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.content[cid]
	if !ok {
		return nil, web3.ErrNotFound("content", nil)
	}
	return append([]byte(nil), data...), nil
}

// GetJSON retrieves and unmarshals JSON by CID.
func (s *Store) GetJSON(ctx context.Context, cid string, result interface{}) error {
	data, err := s.Get(ctx, cid)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, result); err != nil {
		return web3.ErrStorageFailed("unmarshal JSON", err)
	}
	return nil
}

// GetURL returns a gateway-style URL for a CID.
func (s *Store) GetURL(cid string) string {
	return fmt.Sprintf("%s/ipfs/%s", s.gatewayURL, cid)
}

// Pin marks a CID as pinned. The CID must already exist.
func (s *Store) Pin(ctx context.Context, cid string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.content[cid]; !ok {
		return web3.ErrNotFound("content", nil)
	}
	s.pins[cid] = struct{}{}
	return nil
}

// Unpin removes a pin (no-op if not pinned).
func (s *Store) Unpin(ctx context.Context, cid string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pins, cid)
	return nil
}

// ListPins returns all pinned CIDs.
func (s *Store) ListPins(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	pins := make([]string, 0, len(s.pins))
	for cid := range s.pins {
		pins = append(pins, cid)
	}
	return pins, nil
}
