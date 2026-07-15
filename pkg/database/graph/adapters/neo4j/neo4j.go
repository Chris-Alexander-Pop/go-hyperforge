// Package neo4j implements graph.Interface against the Neo4j HTTP transactional API.
package neo4j

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/database/graph"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Store implements graph.Interface using Neo4j's HTTP Cypher API.
type Store struct {
	baseURL  string
	user     string
	password string
	database string
	client   *http.Client
}

// New creates a Neo4j graph store from graph.Config.
// Host may be a full base URL (http://localhost:7474) or hostname with Port.
func New(cfg graph.Config) (*Store, error) {
	base := strings.TrimRight(cfg.Host, "/")
	if base == "" {
		return nil, errors.InvalidArgument("neo4j host is required", nil)
	}
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		port := cfg.Port
		if port == 0 {
			port = 7474
		}
		base = fmt.Sprintf("http://%s:%d", base, port)
	}
	db := cfg.Database
	if db == "" {
		db = "neo4j"
	}
	return &Store{
		baseURL:  base,
		user:     cfg.User,
		password: cfg.Password,
		database: db,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// AddVertex merges a vertex by ID.
func (s *Store) AddVertex(ctx context.Context, v *graph.Vertex) error {
	if v == nil || v.ID == "" {
		return errors.InvalidArgument("vertex id is required", nil)
	}
	label := sanitizeLabel(v.Label)
	props := map[string]interface{}{"id": v.ID}
	for k, val := range v.Properties {
		props[k] = val
	}
	cypher := fmt.Sprintf("MERGE (n:`%s` {id: $id}) SET n += $props RETURN n", label)
	_, err := s.run(ctx, cypher, map[string]interface{}{"id": v.ID, "props": props})
	return err
}

// AddEdge merges an edge between existing vertices.
func (s *Store) AddEdge(ctx context.Context, e *graph.Edge) error {
	if e == nil || e.FromID == "" || e.ToID == "" {
		return errors.InvalidArgument("edge from/to ids are required", nil)
	}
	label := sanitizeLabel(e.Label)
	if label == "" {
		label = "RELATED"
	}
	props := map[string]interface{}{}
	if e.ID != "" {
		props["id"] = e.ID
	}
	for k, val := range e.Properties {
		props[k] = val
	}
	cypher := fmt.Sprintf(`
MATCH (a {id: $from}), (b {id: $to})
MERGE (a)-[r:`+"`%s`"+` {id: $eid}]->(b)
SET r += $props
RETURN r`, label)
	eid := e.ID
	if eid == "" {
		eid = e.FromID + "->" + e.ToID + ":" + label
	}
	_, err := s.run(ctx, cypher, map[string]interface{}{
		"from":  e.FromID,
		"to":    e.ToID,
		"eid":   eid,
		"props": props,
	})
	return err
}

// GetVertex retrieves a vertex by ID.
func (s *Store) GetVertex(ctx context.Context, id string) (*graph.Vertex, error) {
	rows, err := s.run(ctx, "MATCH (n {id: $id}) RETURN n LIMIT 1", map[string]interface{}{"id": id})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.NotFound("vertex not found", nil)
	}
	return nodeToVertex(rows[0]["n"])
}

// GetNeighbors retrieves neighbor vertices by edge label and direction.
func (s *Store) GetNeighbors(ctx context.Context, vertexID string, edgeLabel string, direction string) ([]*graph.Vertex, error) {
	labelFilter := ""
	if edgeLabel != "" {
		labelFilter = ":`" + sanitizeLabel(edgeLabel) + "`"
	}
	var cypher string
	switch direction {
	case "out", "outgoing":
		cypher = fmt.Sprintf("MATCH (n {id: $id})-[%s]->(m) RETURN m", "r"+labelFilter)
	case "in", "incoming":
		cypher = fmt.Sprintf("MATCH (n {id: $id})<-[%s]-(m) RETURN m", "r"+labelFilter)
	default:
		cypher = fmt.Sprintf("MATCH (n {id: $id})-[%s]-(m) RETURN m", "r"+labelFilter)
	}
	rows, err := s.run(ctx, cypher, map[string]interface{}{"id": vertexID})
	if err != nil {
		return nil, err
	}
	out := make([]*graph.Vertex, 0, len(rows))
	for _, row := range rows {
		v, err := nodeToVertex(row["m"])
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// Query executes arbitrary Cypher with parameters.
func (s *Store) Query(ctx context.Context, query string, args map[string]interface{}) (interface{}, error) {
	return s.run(ctx, query, args)
}

// Close is a no-op for the HTTP client.
func (s *Store) Close() error { return nil }

func (s *Store) run(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	if params == nil {
		params = map[string]interface{}{}
	}
	payload := map[string]interface{}{
		"statements": []map[string]interface{}{
			{
				"statement":          cypher,
				"parameters":         params,
				"resultDataContents": []string{"row", "graph"},
			},
		},
	}
	url := fmt.Sprintf("%s/db/%s/tx/commit", s.baseURL, s.database)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal neo4j request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create neo4j request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.user != "" {
		req.SetBasicAuth(s.user, s.password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "neo4j request failed")
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.Internal(fmt.Sprintf("neo4j http %d: %s", resp.StatusCode, string(raw)), nil)
	}

	var parsed struct {
		Results []struct {
			Columns []string `json:"columns"`
			Data    []struct {
				Row []interface{} `json:"row"`
			} `json:"data"`
		} `json:"results"`
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, errors.Wrap(err, "failed to decode neo4j response")
	}
	if len(parsed.Errors) > 0 {
		return nil, errors.Internal("neo4j: "+parsed.Errors[0].Message, nil)
	}
	if len(parsed.Results) == 0 {
		return nil, nil
	}
	res := parsed.Results[0]
	rows := make([]map[string]interface{}, 0, len(res.Data))
	for _, d := range res.Data {
		row := make(map[string]interface{}, len(res.Columns))
		for i, col := range res.Columns {
			if i < len(d.Row) {
				row[col] = d.Row[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func nodeToVertex(raw interface{}) (*graph.Vertex, error) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.Internal("unexpected neo4j node shape", nil)
	}
	v := &graph.Vertex{Properties: map[string]interface{}{}}
	for k, val := range m {
		if k == "id" {
			if s, ok := val.(string); ok {
				v.ID = s
			} else {
				v.ID = fmt.Sprint(val)
			}
			continue
		}
		v.Properties[k] = val
	}
	if v.ID == "" {
		v.ID = fmt.Sprint(m["id"])
	}
	return v, nil
}

func sanitizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "Node"
	}
	var b strings.Builder
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "Node"
	}
	return b.String()
}

var _ graph.Interface = (*Store)(nil)
