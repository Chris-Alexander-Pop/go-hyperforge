// Package memory provides an in-memory embedding.Service for tests.
package memory

import (
	"context"
	"hash/fnv"

	"github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/embedding"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

// Service implements embedding.Service with deterministic pseudo-vectors.
type Service struct {
	dim int
}

// New creates a memory embedding service. Dimension defaults to 8 when <= 0.
func New(dim int) *Service {
	if dim <= 0 {
		dim = 8
	}
	return &Service{dim: dim}
}

// Embed returns deterministic vectors derived from text hashes (test-friendly).
func (s *Service) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(texts) == 0 {
		return nil, errors.InvalidArgument("texts are required", nil)
	}

	out := make([][]float32, len(texts))
	for i, text := range texts {
		out[i] = s.vectorFor(text)
	}
	return out, nil
}

// Dimension returns the vector size.
func (s *Service) Dimension() int {
	return s.dim
}

func (s *Service) vectorFor(text string) []float32 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	seed := h.Sum64()
	vec := make([]float32, s.dim)
	for i := 0; i < s.dim; i++ {
		// Mix seed with index for a stable pseudo-random component in [-1, 1].
		v := float32(((seed>>(i%32))^uint64(i*2654435761))%1000)/500.0 - 1.0
		vec[i] = v
	}
	return vec
}

var _ embedding.Service = (*Service)(nil)
