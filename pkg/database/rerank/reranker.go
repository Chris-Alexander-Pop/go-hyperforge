package rerank

import (
	"context"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/database/vector"
)

// Reranker defines the interface for reranking search results
type Reranker interface {
	Rerank(ctx context.Context, queryVector []float32, candidates []vector.Result) ([]vector.Result, error)
}

// SimpleScorer implements Reranker using field weights
// In reality, this would call an LLM or Cross-Encoder model.
// TBI: Implement actual LLM call (e.g. OpenAI/Cohere API or local ONNX model) - This TBI remains as future roadmap.
type SimpleScorer struct {
	Weights           map[string]float32 // Field Name -> Weight to add/multiply
	DefaultMultiplier float32
}

func NewSimpleScorer(weights map[string]float32) *SimpleScorer {
	if weights == nil {
		weights = make(map[string]float32)
	}
	return &SimpleScorer{Weights: weights, DefaultMultiplier: 1.0}
}

func (s *SimpleScorer) Rerank(ctx context.Context, queryVector []float32, candidates []vector.Result) ([]vector.Result, error) {
	reranked := make([]vector.Result, len(candidates))
	copy(reranked, candidates)

	for i := range reranked {
		score := reranked[i].Score

		// Apply Global Multiplier
		score *= s.DefaultMultiplier

		// Apply Metadata Weights
		// If a metadata key exists in Weights, we add the weight to the score (Boosting)
		if reranked[i].Metadata != nil {
			for key := range reranked[i].Metadata {
				if w, ok := s.Weights[key]; ok {
					score += w // Boost score by weight
				}
			}
		}

		reranked[i].Score = score

		if reranked[i].Metadata == nil {
			reranked[i].Metadata = make(map[string]interface{})
		}
		reranked[i].Metadata["reranked"] = true
	}

	return reranked, nil
}
