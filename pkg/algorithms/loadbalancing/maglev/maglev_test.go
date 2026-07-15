package maglev_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/maglev"
)

func TestMaglevDeterministic(t *testing.T) {
	b := maglev.New(17, "a", "b", "c")
	ctx := context.Background()

	first, err := b.NextKey(ctx, "user-42")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := b.NextKey(ctx, "user-42")
		if err != nil {
			t.Fatal(err)
		}
		if got != first {
			t.Fatalf("expected sticky %s, got %s", first, got)
		}
	}
}

func TestMaglevMinimalChurn(t *testing.T) {
	const size = 101
	b := maglev.New(size, "n1", "n2", "n3", "n4")
	ctx := context.Background()

	before := map[string]string{}
	for i := 0; i < 200; i++ {
		key := string(rune('A'+i%26)) + string(rune('0'+i%10))
		n, err := b.NextKey(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		before[key] = n
	}

	b.Remove("n3")
	moved := 0
	for key, prev := range before {
		n, err := b.NextKey(ctx, key)
		if err != nil {
			t.Fatal(err)
		}
		if n != prev {
			moved++
		}
	}
	// Maglev aims for ~1/N churn; allow generous bound for tiny table.
	if moved > len(before)*3/4 {
		t.Fatalf("too much churn after remove: %d/%d", moved, len(before))
	}
}

func TestMaglevEmpty(t *testing.T) {
	b := maglev.New(7)
	_, err := b.Next(context.Background())
	if err != loadbalancing.ErrNoNodes {
		t.Fatalf("expected ErrNoNodes, got %v", err)
	}
	b.Add("x", 1)
	n, err := b.NextKey(context.Background(), "k")
	if err != nil || n != "x" {
		t.Fatalf("got %q %v", n, err)
	}
}
