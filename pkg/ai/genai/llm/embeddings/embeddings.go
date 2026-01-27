// Package embeddings provides text embedding generation.
package embeddings

import "context"

// Embedder generates vector embeddings for text.
type Embedder interface {
	// EmbedDocuments embeds a list of texts (e.g., for storage).
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery embeds a single query text (e.g., for search).
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
