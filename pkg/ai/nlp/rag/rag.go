// Package rag provides a RAG (Retrieval Augmented Generation) orchestrator
// wired to pkg/database/vector and optionally pkg/database/rerank.
package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/embedding"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/rerank"
	"github.com/chris-alexander-pop/system-design-library/pkg/database/vector"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Orchestrator manages the RAG pipeline: embed → vector search → optional rerank.
type Orchestrator struct {
	embedder    embedding.Service
	vectorStore vector.Store
	reranker    rerank.Reranker
}

// Option configures the orchestrator.
type Option func(*Orchestrator)

// WithReranker attaches a reranker applied after vector search.
func WithReranker(r rerank.Reranker) Option {
	return func(o *Orchestrator) { o.reranker = r }
}

// New creates a new RAG orchestrator.
func New(embedder embedding.Service, store vector.Store, opts ...Option) *Orchestrator {
	o := &Orchestrator{
		embedder:    embedder,
		vectorStore: store,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// Ingest adds document text to the knowledge base.
func (o *Orchestrator) Ingest(ctx context.Context, id, text string, metadata map[string]interface{}) error {
	if o.embedder == nil || o.vectorStore == nil {
		return errors.InvalidArgument("embedder and vector store are required", nil)
	}
	if strings.TrimSpace(text) == "" {
		return errors.InvalidArgument("text is required", nil)
	}

	chunks := strings.Split(text, "\n\n")
	chunkIdx := 0
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk) == "" {
			continue
		}

		vectors, err := o.embedder.Embed(ctx, []string{chunk})
		if err != nil {
			return errors.Wrap(err, "rag embed failed")
		}

		meta := make(map[string]interface{})
		for k, v := range metadata {
			meta[k] = v
		}
		meta["text"] = chunk

		chunkID := id
		if chunkIdx > 0 {
			chunkID = fmt.Sprintf("%s#%d", id, chunkIdx)
		}
		chunkIdx++

		if err := o.vectorStore.Upsert(ctx, chunkID, vectors[0], meta); err != nil {
			return errors.Wrap(err, "rag upsert failed")
		}
	}

	return nil
}

// Retrieve finds relevant context for a query.
func (o *Orchestrator) Retrieve(ctx context.Context, query string, k int) ([]string, error) {
	results, err := o.RetrieveResults(ctx, query, k, nil)
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

// RetrieveResults embeds the query, searches the vector store (with optional
// metadata filter), and optionally reranks candidates.
func (o *Orchestrator) RetrieveResults(ctx context.Context, query string, k int, filter map[string]interface{}) ([]vector.Result, error) {
	if o.embedder == nil || o.vectorStore == nil {
		return nil, errors.InvalidArgument("embedder and vector store are required", nil)
	}
	if k <= 0 {
		k = 5
	}

	vectors, err := o.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, errors.Wrap(err, "rag embed query failed")
	}

	fetch := k
	if o.reranker != nil {
		fetch = k * 3
		if fetch < k {
			fetch = k
		}
	}

	var results []vector.Result
	if len(filter) > 0 {
		results, err = o.vectorStore.SearchWithOpts(ctx, vectors[0], vector.SearchOpts{
			Limit:  fetch,
			Filter: filter,
		})
	} else {
		results, err = o.vectorStore.Search(ctx, vectors[0], fetch)
	}
	if err != nil {
		return nil, errors.Wrap(err, "rag vector search failed")
	}

	if o.reranker != nil && len(results) > 0 {
		results, err = o.reranker.Rerank(ctx, vectors[0], results)
		if err != nil {
			return nil, errors.Wrap(err, "rag rerank failed")
		}
	}

	if len(results) > k {
		results = results[:k]
	}
	return results, nil
}
