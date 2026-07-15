package neptune_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/graph/adapters/neptune"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_RequiresHost(t *testing.T) {
	_, err := neptune.New(graph.Config{})
	require.Error(t, err)
}

func TestNew_BuildsHTTPSURL(t *testing.T) {
	s, err := neptune.New(graph.Config{Host: "cluster.example.com", Port: 8182})
	require.NoError(t, err)
	defer s.Close()
	var _ graph.Interface = s
}

func TestNeptune_AddGetVertex(t *testing.T) {
	var mu sync.Mutex
	var lastBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/gremlin", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		var body map[string]interface{}
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		lastBody = body
		mu.Unlock()

		gremlin, _ := body["gremlin"].(string)
		if strings.Contains(gremlin, "valueMap") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": map[string]interface{}{"code": 200},
				"result": map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"id":    []interface{}{"v1"},
							"label": []interface{}{"Person"},
							"name":  []interface{}{"Ada"},
						},
					},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": map[string]interface{}{"code": 200},
			"result": map[string]interface{}{"data": []interface{}{}},
		})
	}))
	t.Cleanup(srv.Close)

	s, err := neptune.NewFromClient(srv.URL, srv.Client())
	require.NoError(t, err)

	err = s.AddVertex(context.Background(), &graph.Vertex{
		ID:         "v1",
		Label:      "Person",
		Properties: map[string]interface{}{"name": "Ada"},
	})
	require.NoError(t, err)

	mu.Lock()
	require.NotNil(t, lastBody["gremlin"])
	mu.Unlock()

	v, err := s.GetVertex(context.Background(), "v1")
	require.NoError(t, err)
	assert.Equal(t, "v1", v.ID)
	assert.Equal(t, "Person", v.Label)
	assert.Equal(t, "Ada", v.Properties["name"])
}

func TestNeptune_GetVertexNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": map[string]interface{}{"code": 200},
			"result": map[string]interface{}{"data": []interface{}{}},
		})
	}))
	t.Cleanup(srv.Close)

	s, err := neptune.NewFromClient(srv.URL, srv.Client())
	require.NoError(t, err)
	_, err = s.GetVertex(context.Background(), "missing")
	require.Error(t, err)
}

func TestNeptune_Query(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": map[string]interface{}{"code": 200},
			"result": map[string]interface{}{"data": []interface{}{1, 2, 3}},
		})
	}))
	t.Cleanup(srv.Close)

	s, err := neptune.NewFromClient(srv.URL, srv.Client())
	require.NoError(t, err)
	out, err := s.Query(context.Background(), "g.V().count()", nil)
	require.NoError(t, err)
	list, ok := out.([]interface{})
	require.True(t, ok)
	assert.Len(t, list, 3)
}

func TestNewFromClient_RequiresArgs(t *testing.T) {
	_, err := neptune.NewFromClient("", nil)
	require.Error(t, err)
	_, err = neptune.NewFromClient("http://x", nil)
	require.Error(t, err)
}
