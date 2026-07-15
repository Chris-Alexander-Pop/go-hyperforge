package opensearch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search/adapters/opensearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSearchHTTP_CRUD(t *testing.T) {
	indexes := map[string]map[string]map[string]interface{}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && !strings.Contains(r.URL.Path, "/_doc"):
			name := strings.Trim(r.URL.Path, "/")
			if _, ok := indexes[name]; ok {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"type":"resource_already_exists_exception"}}`)
				return
			}
			indexes[name] = map[string]map[string]interface{}{}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"acknowledged": true})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/_doc/"):
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			name, id := parts[0], parts[2]
			var doc map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&doc)
			indexes[name][id] = doc
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"_id": id, "result": "created"})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/_doc/"):
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			name, id := parts[0], parts[2]
			doc, ok := indexes[name][id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"_id": id, "_source": doc})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/_search"):
			name := strings.TrimSuffix(strings.Trim(r.URL.Path, "/"), "/_search")
			hits := []map[string]interface{}{}
			for id, doc := range indexes[name] {
				hits = append(hits, map[string]interface{}{
					"_id": id, "_score": 1.0, "_source": doc,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"took": 1,
				"hits": map[string]interface{}{
					"total":     map[string]interface{}{"value": len(hits)},
					"max_score": 1.0,
					"hits":      hits,
				},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/_stats"):
			name := strings.TrimSuffix(strings.Trim(r.URL.Path, "/"), "/_stats")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"indices": map[string]interface{}{
					name: map[string]interface{}{
						"primaries": map[string]interface{}{
							"docs":  map[string]interface{}{"count": len(indexes[name])},
							"store": map[string]interface{}{"size_in_bytes": 100},
						},
					},
				},
			})
		case r.Method == http.MethodDelete && !strings.Contains(r.URL.Path, "/_doc"):
			name := strings.Trim(r.URL.Path, "/")
			delete(indexes, name)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"acknowledged": true})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	eng, err := opensearch.New(search.Config{OpenSearchURL: srv.URL})
	require.NoError(t, err)
	eng.WithHTTPClient(srv.Client())

	ctx := context.Background()
	require.NoError(t, eng.CreateIndex(ctx, "products", nil))
	require.NoError(t, eng.Index(ctx, "products", "1", map[string]interface{}{"title": "Phone"}))

	hit, err := eng.Get(ctx, "products", "1")
	require.NoError(t, err)
	assert.Equal(t, "Phone", hit.Source["title"])

	res, err := eng.Search(ctx, "products", search.Query{Text: "Phone"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)

	info, err := eng.GetIndex(ctx, "products")
	require.NoError(t, err)
	assert.Equal(t, int64(1), info.DocCount)

	require.NoError(t, eng.DeleteIndex(ctx, "products"))
}

func TestOpenSearch_RequiresURL(t *testing.T) {
	_, err := opensearch.New(search.Config{})
	require.Error(t, err)
}
