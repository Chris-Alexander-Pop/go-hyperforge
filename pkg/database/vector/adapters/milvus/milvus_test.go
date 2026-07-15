package milvus_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector/adapters/milvus"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

type milvusBackend struct {
	mu      sync.Mutex
	vectors map[string]struct {
		vec  []float32
		meta map[string]interface{}
	}
}

func newBackend() *milvusBackend {
	return &milvusBackend{vectors: make(map[string]struct {
		vec  []float32
		meta map[string]interface{}
	})}
}

func (b *milvusBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var payload map[string]interface{}
	_ = json.Unmarshal(body, &payload)

	b.mu.Lock()
	defer b.mu.Unlock()

	switch {
	case strings.HasSuffix(r.URL.Path, "/entities/upsert"):
		data, _ := payload["data"].([]interface{})
		for _, row := range data {
			m, _ := row.(map[string]interface{})
			id := fmtSprint(m["id"])
			var vec []float32
			if raw, ok := m["vector"].([]interface{}); ok {
				for _, v := range raw {
					if f, ok := v.(float64); ok {
						vec = append(vec, float32(f))
					}
				}
			}
			meta := map[string]interface{}{}
			for k, v := range m {
				if k == "id" || k == "vector" {
					continue
				}
				meta[k] = v
			}
			b.vectors[id] = struct {
				vec  []float32
				meta map[string]interface{}
			}{vec: vec, meta: meta}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0})
	case strings.HasSuffix(r.URL.Path, "/entities/search"):
		limit := 10
		if lim, ok := payload["limit"].(float64); ok {
			limit = int(lim)
		}
		filter, _ := payload["filter"].(string)
		type hit struct {
			ID       string                 `json:"id"`
			Score    float32                `json:"score"`
			Distance float32                `json:"distance"`
			Entity   map[string]interface{} `json:"entity"`
		}
		var hits []hit
		for id, ent := range b.vectors {
			if filter != "" && !matchFilter(filter, ent.meta) {
				continue
			}
			entity := map[string]interface{}{}
			for k, v := range ent.meta {
				entity[k] = v
			}
			hits = append(hits, hit{ID: id, Score: 0.9, Distance: 0.1, Entity: entity})
			if len(hits) >= limit {
				break
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": hits})
	case strings.HasSuffix(r.URL.Path, "/entities/delete"):
		filter, _ := payload["filter"].(string)
		// filter like: id == "x"
		id := extractQuoted(filter)
		if _, ok := b.vectors[id]; !ok {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 404, "message": "not found"})
			return
		}
		delete(b.vectors, id)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0})
	default:
		http.NotFound(w, r)
	}
}

func fmtSprint(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(toJSON(v), "\"", ""), "\n", ""))
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func extractQuoted(s string) string {
	i := strings.Index(s, `"`)
	if i < 0 {
		return ""
	}
	j := strings.LastIndex(s, `"`)
	if j <= i {
		return ""
	}
	return s[i+1 : j]
}

func matchFilter(filter string, meta map[string]interface{}) bool {
	// "source == \"docs\""
	parts := strings.Split(filter, "&&")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		kv := strings.SplitN(p, "==", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		want := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		got := fmtSprint(meta[key])
		if got != want {
			return false
		}
	}
	return true
}

func TestNewRequiresHost(t *testing.T) {
	_, err := milvus.New(vector.Config{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpsertSearchDelete(t *testing.T) {
	backend := newBackend()
	srv := httptest.NewServer(backend)
	defer srv.Close()

	store, err := milvus.New(vector.Config{
		Host:      srv.URL,
		IndexName: "docs",
		APIKey:    "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	store.WithHTTPClient(srv.Client())
	defer store.Close()

	ctx := context.Background()
	vec := []float32{0.1, 0.2, 0.3}
	if err := store.Upsert(ctx, "v1", vec, map[string]interface{}{"source": "docs"}); err != nil {
		t.Fatal(err)
	}

	results, err := store.Search(ctx, vec, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].ID != "v1" {
		t.Fatalf("search: %+v", results)
	}

	filtered, err := store.SearchWithOpts(ctx, vec, vector.SearchOpts{
		Limit:  5,
		Filter: map[string]interface{}{"source": "docs"},
	})
	if err != nil || len(filtered) != 1 {
		t.Fatalf("filtered: %v %+v", err, filtered)
	}

	if err := store.Delete(ctx, "v1"); err != nil {
		t.Fatal(err)
	}
	err = store.Delete(ctx, "missing")
	if err == nil || !errors.Is(err, vector.ErrVectorNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestImplementsStore(t *testing.T) {
	s, err := milvus.New(vector.Config{Host: "http://localhost"})
	if err != nil {
		t.Fatal(err)
	}
	var _ vector.Store = s
}
