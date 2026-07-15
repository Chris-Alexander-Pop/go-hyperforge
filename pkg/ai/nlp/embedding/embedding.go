// Package embedding provides an interface for text embeddings.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/ai/nlp/embedding"
//
//	service := embedding.New(config)
//	vectors, err := service.Embed(ctx, []string{"hello world"})
package embedding

import "context"

// Service is the interface for embedding generation.
type Service interface {
	// Embed generates vectors for a list of texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// Dimension returns the vector dimension.
	Dimension() int
}

// Config holds common embedding configuration.
type Config struct {
	Model     string
	Dimension int
	BatchSize int
}
