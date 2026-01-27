// Package rag provides Retrieval Augmented Generation services.
package rag

import (
	"context"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/genai/llm/embeddings"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/vector"
)

// Orchestrator manages Document Ingestion and Retrieval.
type Orchestrator struct {
	embedder    embeddings.Embedder
	vectorStore vector.Store
}

// New creates a new RAG orchestrator.
func New(embedder embeddings.Embedder, store vector.Store) *Orchestrator {
	return &Orchestrator{
		embedder:    embedder,
		vectorStore: store,
	}
}

// Ingest adds document to vector store.
func (o *Orchestrator) Ingest(ctx context.Context, id, text string, metadata map[string]interface{}) error {
	// Simple split
	chunks := strings.Split(text, "\n\n")

	for i, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		vec, err := o.embedder.EmbedQuery(ctx, chunk)
		if err != nil {
			return err
		}

		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		// Copy metadata to avoid race/overlap
		chunkMeta := make(map[string]interface{})
		for k, v := range metadata {
			chunkMeta[k] = v
		}
		chunkMeta["text"] = chunk
		chunkMeta["chunk_index"] = i

		// ID generation strategy needed
		err = o.vectorStore.Upsert(ctx, id, vec, chunkMeta)
		if err != nil {
			return err
		}
	}
	return nil
}

// Retrieve context for query.
func (o *Orchestrator) Retrieve(ctx context.Context, query string, k int) ([]string, error) {
	vec, err := o.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	results, err := o.vectorStore.Search(ctx, vec, k)
	if err != nil {
		return nil, err
	}

	contexts := make([]string, 0, len(results))
	for _, res := range results {
		if txt, ok := res.Metadata["text"].(string); ok {
			contexts = append(contexts, txt)
		}
	}
	return contexts, nil
}
