package typesense_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/data/search/adapters/typesense"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypesenseHTTP_IndexSearchGet(t *testing.T) {
	collections := map[string]map[string]map[string]interface{}{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("X-TYPESENSE-API-KEY"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/collections":
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name, _ := body["name"].(string)
			if _, ok := collections[name]; ok {
				w.WriteHeader(http.StatusConflict)
				return
			}
			collections[name] = map[string]map[string]interface{}{}
			_ = json.NewEncoder(w).Encode(body)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/collections/") && !strings.Contains(r.URL.Path, "/documents"):
			name := strings.TrimPrefix(r.URL.Path, "/collections/")
			if _, ok := collections[name]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name": name, "num_documents": len(collections[name]), "created_at": 1700000000,
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/documents") && !strings.Contains(r.URL.Path, "/search") && !strings.Contains(r.URL.Path, "/import"):
			parts := strings.Split(r.URL.Path, "/")
			name := parts[2]
			var doc map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&doc)
			id, _ := doc["id"].(string)
			collections[name][id] = doc
			_ = json.NewEncoder(w).Encode(doc)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/documents/") && !strings.Contains(r.URL.Path, "/search"):
			parts := strings.Split(r.URL.Path, "/")
			name, id := parts[2], parts[4]
			doc, ok := collections[name][id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(doc)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/documents/search"):
			parts := strings.Split(r.URL.Path, "/")
			name := parts[2]
			hits := []map[string]interface{}{}
			for id, doc := range collections[name] {
				hits = append(hits, map[string]interface{}{
					"document": doc, "text_match": 100.0, "id": id,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"found": len(hits), "search_time_ms": 2, "hits": hits,
			})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/collections/"):
			name := strings.TrimPrefix(r.URL.Path, "/collections/")
			delete(collections, name)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	eng, err := typesense.New(search.Config{TypesenseURL: srv.URL, TypesenseAPIKey: "test-key"})
	require.NoError(t, err)
	eng.WithHTTPClient(srv.Client())

	ctx := context.Background()
	require.NoError(t, eng.CreateIndex(ctx, "products", &search.IndexMapping{
		Fields: map[string]search.FieldMapping{
			"title": {Type: search.FieldTypeText, Searchable: true},
		},
	}))
	require.NoError(t, eng.Index(ctx, "products", "1", map[string]interface{}{"title": "Laptop"}))

	hit, err := eng.Get(ctx, "products", "1")
	require.NoError(t, err)
	assert.Equal(t, "1", hit.ID)
	assert.Equal(t, "Laptop", hit.Source["title"])

	res, err := eng.Search(ctx, "products", search.Query{Text: "Laptop", Fields: []string{"title"}})
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	require.Len(t, res.Hits, 1)

	info, err := eng.GetIndex(ctx, "products")
	require.NoError(t, err)
	assert.Equal(t, int64(1), info.DocCount)

	require.NoError(t, eng.DeleteIndex(ctx, "products"))
	require.NoError(t, eng.Close())
}

func TestTypesense_RequiresURL(t *testing.T) {
	_, err := typesense.New(search.Config{})
	require.Error(t, err)
}
