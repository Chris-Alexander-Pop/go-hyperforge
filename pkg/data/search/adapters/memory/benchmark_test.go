package memory

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/data/search"
)

func BenchmarkSearch(b *testing.B) {
	ctx := context.Background()
	engine := New()

	// Create index
	err := engine.CreateIndex(ctx, "bench-index", nil)
	if err != nil {
		b.Fatalf("failed to create index: %v", err)
	}

	// Generate documents
	// We use a small vocabulary to ensure some collisions and meaningful search
	vocab := []string{
		"apple", "banana", "cherry", "date", "elderberry", "fig", "grape", "honeydew",
		"iguana", "jackfruit", "kiwi", "lemon", "mango", "nectarine", "orange", "papaya",
		"quince", "raspberry", "strawberry", "tangerine", "ugli", "vanilla", "watermelon",
		"xigua", "yuzu", "zucchini", "laptop", "computer", "phone", "mobile", "server",
		"cloud", "network", "database", "algorithm", "structure", "design", "system",
	}

	numDocs := 1000
	for i := 0; i < numDocs; i++ {
		// Create a random sentence
		wordCount := 5 + rand.Intn(10)
		sentence := ""
		for j := 0; j < wordCount; j++ {
			sentence += vocab[rand.Intn(len(vocab))] + " "
		}

		doc := map[string]interface{}{
			"title": fmt.Sprintf("Document %d", i),
			"body":  sentence,
			"tags":  []string{vocab[rand.Intn(len(vocab))], vocab[rand.Intn(len(vocab))]},
		}

		if err := engine.Index(ctx, "bench-index", fmt.Sprintf("doc-%d", i), doc); err != nil {
			b.Fatalf("failed to index doc: %v", err)
		}
	}

	b.ResetTimer()

	// Queries to benchmark
	queries := []string{
		"apple",
		"computer",
		"system",
		"banana",
	}

	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		result, err := engine.Search(ctx, "bench-index", search.Query{
			Text: q,
		})
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
		if result.Total == 0 {
			b.Fatalf("expected results for query %q, got 0", q)
		}
	}
}
