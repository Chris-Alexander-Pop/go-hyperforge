package vector

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// HybridOpts configures hybrid keyword + vector search.
type HybridOpts struct {
	// Limit is the maximum number of hybrid results (default 10).
	Limit int

	// KeywordQuery is matched against metadata string values (case-insensitive substring).
	// Empty skips keyword scoring (vector-only with optional filter).
	KeywordQuery string

	// Filter is an exact-match metadata filter applied before scoring (same as SearchWithOpts).
	Filter map[string]interface{}

	// KeywordWeight multiplies the keyword score (default 0.3).
	KeywordWeight float32

	// VectorWeight multiplies the vector similarity score (default 0.7).
	VectorWeight float32

	// CandidateLimit is how many vector neighbors to fetch before hybrid re-rank
	// (default max(Limit*5, 50)).
	CandidateLimit int
}

// HybridResult is a search hit with decomposed scores.
type HybridResult struct {
	ID           string                 `json:"id"`
	Metadata     map[string]interface{} `json:"metadata"`
	KeywordScore float32                `json:"keyword_score"`
	VectorScore  float32                `json:"vector_score"`
	HybridScore  float32                `json:"hybrid_score"`
}

// HybridSearch combines vector similarity with keyword metadata scoring.
//
// KeywordScore is 1.0 when KeywordQuery is empty (neutral), otherwise the fraction
// of metadata string fields that contain the query (0..1), boosted to 1.0 if any
// field equals the query exactly (case-insensitive).
//
// HybridScore = KeywordWeight*KeywordScore + VectorWeight*VectorScore
// (weights are normalized to sum to 1 when both are positive).
func HybridSearch(ctx context.Context, store Store, query []float32, opts HybridOpts) ([]HybridResult, error) {
	if store == nil {
		return nil, errors.InvalidArgument("vector store is required", nil)
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	kwW, vecW := opts.KeywordWeight, opts.VectorWeight
	if kwW == 0 && vecW == 0 {
		kwW, vecW = 0.3, 0.7
	}
	sum := kwW + vecW
	if sum > 0 {
		kwW /= sum
		vecW /= sum
	}
	cand := opts.CandidateLimit
	if cand <= 0 {
		cand = limit * 5
		if cand < 50 {
			cand = 50
		}
	}

	vectorHits, err := store.SearchWithOpts(ctx, query, SearchOpts{
		Limit:  cand,
		Filter: opts.Filter,
	})
	if err != nil {
		return nil, err
	}

	q := strings.TrimSpace(strings.ToLower(opts.KeywordQuery))
	out := make([]HybridResult, 0, len(vectorHits))
	for _, hit := range vectorHits {
		kw := float32(1)
		if q != "" {
			kw = keywordScore(hit.Metadata, q)
		}
		hybrid := kwW*kw + vecW*hit.Score
		out = append(out, HybridResult{
			ID:           hit.ID,
			Metadata:     hit.Metadata,
			KeywordScore: kw,
			VectorScore:  hit.Score,
			HybridScore:  hybrid,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].HybridScore == out[j].HybridScore {
			return out[i].ID < out[j].ID
		}
		return out[i].HybridScore > out[j].HybridScore
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func keywordScore(metadata map[string]interface{}, queryLower string) float32 {
	if len(metadata) == 0 {
		return 0
	}
	var matched, total float32
	exact := false
	for _, v := range metadata {
		s, ok := v.(string)
		if !ok {
			continue
		}
		total++
		lower := strings.ToLower(s)
		if lower == queryLower {
			exact = true
			matched++
			continue
		}
		if strings.Contains(lower, queryLower) {
			matched++
		}
	}
	if total == 0 {
		return 0
	}
	if exact {
		return 1
	}
	score := matched / total
	return float32(math.Min(1, float64(score)))
}
