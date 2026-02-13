package memory

import (
	"context"
	"math"
	"sort"

	"github.com/chris-alexander-pop/system-design-library/pkg/concurrency"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/vector"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

type entry struct {
	id       string
	vector   []float32
	metadata map[string]interface{}
}

// Store implements vector.Store with an in-memory store.
type Store struct {
	entries []entry
	mu      *concurrency.SmartRWMutex
}

// New creates a new in-memory vector store.
func New() *Store {
	return &Store{
		entries: make([]entry, 0),
		mu:      concurrency.NewSmartRWMutex(concurrency.MutexConfig{Name: "memory-vector"}),
	}
}

// Search finds the nearest neighbors using cosine similarity.
func (s *Store) Search(ctx context.Context, queryVector []float32, limit int) ([]vector.Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.entries) == 0 {
		return []vector.Result{}, nil
	}

	type scored struct {
		entry entry
		score float32
	}

	results := make([]scored, 0, len(s.entries))
	for _, e := range s.entries {
		score := cosineSimilarity(queryVector, e.vector)
		results = append(results, scored{entry: e, score: score})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if limit > len(results) {
		limit = len(results)
	}

	output := make([]vector.Result, limit)
	for i := 0; i < limit; i++ {
		output[i] = vector.Result{
			ID:       results[i].entry.id,
			Score:    results[i].score,
			Metadata: cloneMetadata(results[i].entry.metadata),
		}
	}

	return output, nil
}

// Upsert inserts or updates a vector.
func (s *Store) Upsert(ctx context.Context, id string, vec []float32, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if ID exists, update if so
	for i, e := range s.entries {
		if e.id == id {
			s.entries[i].vector = cloneVector(vec)
			s.entries[i].metadata = cloneMetadata(metadata)
			return nil
		}
	}

	// Insert new
	s.entries = append(s.entries, entry{
		id:       id,
		vector:   cloneVector(vec),
		metadata: cloneMetadata(metadata),
	})
	return nil
}

// Delete removes a vector by ID.
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.entries {
		if e.id == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return nil
		}
	}
	return errors.NotFound("vector not found", nil)
}

// Close clears the in-memory store.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func cloneVector(v []float32) []float32 {
	c := make([]float32, len(v))
	copy(c, v)
	return c
}

func cloneMetadata(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	c := make(map[string]interface{})
	for k, v := range m {
		c[k] = v
	}
	return c
}

// Ensure Store implements vector.Store
var _ vector.Store = (*Store)(nil)
