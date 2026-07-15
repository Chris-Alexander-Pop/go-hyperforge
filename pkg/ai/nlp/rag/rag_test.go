package rag_test

import (
	"context"
	"testing"

	embedmem "github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/embedding/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/rag"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/rerank"
	vecmem "github.com/chris-alexander-pop/system-design-library/pkg/database/vector/adapters/memory"
)

func TestRAG_RetrieveWithRerankAndFilter(t *testing.T) {
	ctx := context.Background()
	embedder := embedmem.New(8)
	store := vecmem.New()
	scorer := rerank.NewSimpleScorer(map[string]float32{"boost": 10})

	orch := rag.New(embedder, store, rag.WithReranker(scorer))

	if err := orch.Ingest(ctx, "doc1", "Cats are mammals.\n\nDogs bark loudly.", map[string]interface{}{
		"source": "pets",
		"boost":  true,
	}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if err := orch.Ingest(ctx, "doc2", "Quantum physics is complex.", map[string]interface{}{
		"source": "science",
	}); err != nil {
		t.Fatalf("Ingest2: %v", err)
	}

	results, err := orch.RetrieveResults(ctx, "mammals and cats", 5, map[string]interface{}{"source": "pets"})
	if err != nil {
		t.Fatalf("RetrieveResults: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected filtered results")
	}
	for _, r := range results {
		if r.Metadata["source"] != "pets" {
			t.Fatalf("filter leak: %+v", r)
		}
		if r.Metadata["reranked"] != true {
			t.Fatalf("expected reranked metadata: %+v", r)
		}
	}

	texts, err := orch.Retrieve(ctx, "cats", 3)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(texts) == 0 {
		t.Fatal("expected text contexts")
	}
}
