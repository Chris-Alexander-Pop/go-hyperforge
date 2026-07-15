// Package neptune implements graph.Interface against Amazon Neptune's Gremlin HTTP API.
//
// Neptune accepts Gremlin over HTTPS at /gremlin. Tests inject an HTTP client
// via NewFromClient / WithHTTPClient (httptest); production uses New(cfg).
package neptune

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Doer is the HTTP surface used by the adapter (*http.Client satisfies it).
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Store implements graph.Interface using Neptune Gremlin over HTTP.
type Store struct {
	baseURL  string
	user     string
	password string
	client   Doer
}

// New creates a Neptune graph store from graph.Config.
// Host may be a full base URL (https://cluster.region.neptune.amazonaws.com:8182)
// or hostname with Port (default 8182).
func New(cfg graph.Config) (*Store, error) {
	base := strings.TrimRight(cfg.Host, "/")
	if base == "" {
		return nil, errors.InvalidArgument("neptune host is required", nil)
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		port := cfg.Port
		if port == 0 {
			port = 8182
		}
		base = fmt.Sprintf("https://%s:%d", base, port)
	}
	return &Store{
		baseURL:  base,
		user:     cfg.User,
		password: cfg.Password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// NewFromClient wraps an existing Doer (production client or test double).
func NewFromClient(baseURL string, client Doer) (*Store, error) {
	base := strings.TrimRight(baseURL, "/")
	if base == "" {
		return nil, errors.InvalidArgument("neptune base URL is required", nil)
	}
	if client == nil {
		return nil, errors.InvalidArgument("neptune http client is required", nil)
	}
	return &Store{baseURL: base, client: client}, nil
}

// WithHTTPClient overrides the HTTP Doer (tests / custom transport).
func (s *Store) WithHTTPClient(c Doer) *Store {
	if c != nil {
		s.client = c
	}
	return s
}

var _ graph.Interface = (*Store)(nil)

// AddVertex upserts a vertex by ID with optional label and properties.
func (s *Store) AddVertex(ctx context.Context, v *graph.Vertex) error {
	if v == nil || v.ID == "" {
		return errors.InvalidArgument("vertex id is required", nil)
	}
	label := sanitizeLabel(v.Label)
	gremlin := `g.V().has('id', id).fold().coalesce(unfold(), addV(label).property('id', id))`
	bindings := map[string]interface{}{"id": v.ID, "label": label}
	for k, val := range v.Properties {
		key := sanitizeProp(k)
		if key == "" || key == "id" {
			continue
		}
		gremlin += fmt.Sprintf(".property('%s', p_%s)", key, key)
		bindings["p_"+key] = val
	}
	_, err := s.run(ctx, gremlin, bindings)
	return err
}

// AddEdge creates an edge between two vertices identified by property id.
func (s *Store) AddEdge(ctx context.Context, e *graph.Edge) error {
	if e == nil || e.FromID == "" || e.ToID == "" {
		return errors.InvalidArgument("edge from/to ids are required", nil)
	}
	label := sanitizeLabel(e.Label)
	if label == "" {
		label = "RELATED"
	}
	eid := e.ID
	if eid == "" {
		eid = e.FromID + "->" + e.ToID + ":" + label
	}
	gremlin := `g.V().has('id', from).as('a').V().has('id', to).as('b').addE(label).from('a').to('b').property('id', eid)`
	bindings := map[string]interface{}{
		"from":  e.FromID,
		"to":    e.ToID,
		"label": label,
		"eid":   eid,
	}
	for k, val := range e.Properties {
		key := sanitizeProp(k)
		if key == "" || key == "id" {
			continue
		}
		gremlin += fmt.Sprintf(".property('%s', p_%s)", key, key)
		bindings["p_"+key] = val
	}
	_, err := s.run(ctx, gremlin, bindings)
	return err
}

// GetVertex retrieves a vertex by property id.
func (s *Store) GetVertex(ctx context.Context, id string) (*graph.Vertex, error) {
	rows, err := s.run(ctx, `g.V().has('id', id).valueMap(true)`, map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	list, ok := rows.([]interface{})
	if !ok || len(list) == 0 {
		return nil, errors.NotFound("vertex not found", nil)
	}
	return valueMapToVertex(list[0], id)
}

// GetNeighbors retrieves neighbor vertices by edge label and direction.
func (s *Store) GetNeighbors(ctx context.Context, vertexID string, edgeLabel string, direction string) ([]*graph.Vertex, error) {
	var traversal string
	switch direction {
	case "out", "outgoing":
		if edgeLabel != "" {
			traversal = `g.V().has('id', id).out(label).valueMap(true)`
		} else {
			traversal = `g.V().has('id', id).out().valueMap(true)`
		}
	case "in", "incoming":
		if edgeLabel != "" {
			traversal = `g.V().has('id', id).in(label).valueMap(true)`
		} else {
			traversal = `g.V().has('id', id).in().valueMap(true)`
		}
	default:
		if edgeLabel != "" {
			traversal = `g.V().has('id', id).both(label).valueMap(true)`
		} else {
			traversal = `g.V().has('id', id).both().valueMap(true)`
		}
	}
	bindings := map[string]interface{}{"id": vertexID}
	if edgeLabel != "" {
		bindings["label"] = sanitizeLabel(edgeLabel)
	}
	rows, err := s.run(ctx, traversal, bindings)
	if err != nil {
		return nil, err
	}
	list, ok := rows.([]interface{})
	if !ok {
		return nil, nil
	}
	out := make([]*graph.Vertex, 0, len(list))
	for _, item := range list {
		v, err := valueMapToVertex(item, "")
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// Query executes arbitrary Gremlin with optional bindings (args).
func (s *Store) Query(ctx context.Context, query string, args map[string]interface{}) (interface{}, error) {
	return s.run(ctx, query, args)
}

// Close is a no-op for the HTTP client.
func (s *Store) Close() error { return nil }

func (s *Store) run(ctx context.Context, gremlin string, bindings map[string]interface{}) (interface{}, error) {
	if bindings == nil {
		bindings = map[string]interface{}{}
	}
	payload := map[string]interface{}{
		"gremlin":  gremlin,
		"bindings": bindings,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal neptune request")
	}
	url := s.baseURL + "/gremlin"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create neptune request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.user != "" {
		req.SetBasicAuth(s.user, s.password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "neptune request failed")
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Internal(fmt.Sprintf("neptune http %d: %s", resp.StatusCode, string(raw)), nil)
	}

	var parsed struct {
		RequestID string `json:"requestId"`
		Status    struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
		Result struct {
			Data interface{} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to decode neptune response")
	}
	if parsed.Status.Code != 0 && parsed.Status.Code != 200 {
		msg := parsed.Status.Message
		if msg == "" {
			msg = fmt.Sprintf("neptune status %d", parsed.Status.Code)
		}
		return nil, errors.Internal("neptune: "+msg, nil)
	}
	return parsed.Result.Data, nil
}

func valueMapToVertex(raw interface{}, fallbackID string) (*graph.Vertex, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.Internal("unexpected neptune vertex shape", nil)
	}
	v := &graph.Vertex{Properties: map[string]interface{}{}}
	for k, val := range m {
		if k == "id" || k == "T.id" {
			v.ID = firstString(val)
			continue
		}
		if k == "label" || k == "T.label" {
			v.Label = firstString(val)
			continue
		}
		v.Properties[k] = unwrapList(val)
	}
	if v.ID == "" {
		v.ID = fallbackID
	}
	if v.ID == "" {
		return nil, errors.Internal("neptune vertex missing id", nil)
	}
	return v, nil
}

func firstString(val interface{}) string {
	switch t := val.(type) {
	case string:
		return t
	case []interface{}:
		if len(t) > 0 {
			return fmt.Sprint(t[0])
		}
	case float64:
		return fmt.Sprint(int64(t))
	}
	return fmt.Sprint(val)
}

func unwrapList(val interface{}) interface{} {
	if list, ok := val.([]interface{}); ok {
		if len(list) == 1 {
			return list[0]
		}
		return list
	}
	return val
}

func sanitizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "Vertex"
	}
	return sanitizeProp(label)
}

func sanitizeProp(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
