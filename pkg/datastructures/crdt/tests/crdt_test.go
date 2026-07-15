package crdt_test

import (
	"testing"
	"time"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/crdt"
)

func TestGCounter_IncAndMerge(t *testing.T) {
	a := crdt.NewGCounter("a")
	b := crdt.NewGCounter("b")

	a.Inc(3)
	b.Inc(5)

	if a.Count() != 3 {
		t.Fatalf("a.Count()=%d want 3", a.Count())
	}

	a.Merge(b)
	if a.Count() != 8 {
		t.Fatalf("after merge a.Count()=%d want 8", a.Count())
	}

	// Idempotent merge
	a.Merge(b)
	if a.Count() != 8 {
		t.Fatalf("idempotent merge a.Count()=%d want 8", a.Count())
	}
}

func TestPNCounter_IncDecMerge(t *testing.T) {
	a := crdt.NewPNCounter("a")
	b := crdt.NewPNCounter("b")

	a.Inc(10)
	a.Dec(3)
	b.Inc(2)
	b.Dec(1)

	if a.Count() != 7 {
		t.Fatalf("a.Count()=%d want 7", a.Count())
	}

	a.Merge(b)
	if a.Count() != 8 {
		t.Fatalf("after merge a.Count()=%d want 8", a.Count())
	}
}

func TestLWWRegister_SetAndMerge(t *testing.T) {
	r1 := crdt.NewLWWRegister("n1", "old")
	r2 := crdt.NewLWWRegister("n2", "newer")

	ts := time.Now().UnixNano()
	r1.Set("mid", ts)
	r2.Set("latest", ts+1000)

	r1.Merge(r2)
	if got := r1.Get(); got != "latest" {
		t.Fatalf("Get()=%q want latest", got)
	}

	// Older write must not overwrite
	r1.Set("stale", ts)
	if got := r1.Get(); got != "latest" {
		t.Fatalf("Get()=%q want latest after stale Set", got)
	}
}

func TestGSet_AddContainsMerge(t *testing.T) {
	a := crdt.NewGSet[string]()
	b := crdt.NewGSet[string]()

	a.Add("x")
	a.Add("y")
	b.Add("y")
	b.Add("z")

	if !a.Contains("x") || a.Contains("z") {
		t.Fatal("unexpected Contains before merge")
	}
	if a.Size() != 2 {
		t.Fatalf("Size()=%d want 2", a.Size())
	}

	a.Merge(b)
	if !a.Contains("z") || a.Size() != 3 {
		t.Fatalf("after merge Contains(z)=%v Size=%d", a.Contains("z"), a.Size())
	}

	els := a.Elements()
	if len(els) != 3 {
		t.Fatalf("Elements len=%d want 3", len(els))
	}

	// Grow-only: duplicate add is a no-op
	a.Add("x")
	if a.Size() != 3 {
		t.Fatalf("duplicate Add changed size to %d", a.Size())
	}
}
