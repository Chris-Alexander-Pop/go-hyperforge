package memory_test

import (
	"context"
	"testing"

	embmemory "github.com/chris-alexander-pop/system-design-library/pkg/ai/nlp/embedding/adapters/memory"
	"github.com/chris-alexander-pop/system-design-library/pkg/errors"
)

func TestEmbedDeterministic(t *testing.T) {
	svc := embmemory.New(4)
	ctx := context.Background()

	a, err := svc.Embed(ctx, []string{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Embed(ctx, []string{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || len(a[0]) != 4 {
		t.Fatalf("shape=%v dim=%d", len(a), svc.Dimension())
	}
	for i := range a[0] {
		if a[0][i] != b[0][i] {
			t.Fatalf("not deterministic at %d", i)
		}
	}

	_, err = svc.Embed(ctx, nil)
	if !errors.IsCode(err, errors.CodeInvalidArgument) {
		t.Fatalf("want invalid argument, got %v", err)
	}
}
