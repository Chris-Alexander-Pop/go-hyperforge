package p2c_test

import (
	"context"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/loadbalancing/p2c"
)

func TestP2CPrefersLessLoaded(t *testing.T) {
	b := p2c.NewWithSeed(42, "a", "b")
	ctx := context.Background()

	// Heavily load "a"
	for i := 0; i < 100; i++ {
		b.Inc("a")
	}

	counts := map[string]int{}
	for i := 0; i < 200; i++ {
		n, err := b.Next(ctx)
		if err != nil {
			t.Fatal(err)
		}
		counts[n]++
	}
	if counts["b"] <= counts["a"] {
		t.Fatalf("expected b preferred under load, got %#v", counts)
	}
}

func TestP2CEmpty(t *testing.T) {
	b := p2c.New()
	_, err := b.Next(context.Background())
	if err != loadbalancing.ErrNoNodes {
		t.Fatalf("expected ErrNoNodes, got %v", err)
	}
}

func TestP2CAddRemove(t *testing.T) {
	b := p2c.New("x")
	b.Add("y", 1)
	b.Remove("x")
	n, err := b.Next(context.Background())
	if err != nil || n != "y" {
		t.Fatalf("got %q %v", n, err)
	}
	b.Inc("y")
	if b.Load("y") != 1 {
		t.Fatalf("load=%d", b.Load("y"))
	}
	b.Dec("y")
	if b.Load("y") != 0 {
		t.Fatalf("load=%d", b.Load("y"))
	}
}
