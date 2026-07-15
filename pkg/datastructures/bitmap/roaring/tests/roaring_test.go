package roaring_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/bitmap/roaring"
)

func TestRoaring_AddContains(t *testing.T) {
	b := roaring.New()
	if b.Contains(42) {
		t.Fatal("empty bitmap should not contain 42")
	}
	b.Add(42)
	b.Add(1 << 20) // different high container
	if !b.Contains(42) || !b.Contains(1<<20) {
		t.Fatal("expected Contains after Add")
	}
	if b.Contains(43) {
		t.Fatal("unexpected Contains(43)")
	}
}

func TestRoaring_IdempotentAdd(t *testing.T) {
	b := roaring.New()
	b.Add(7)
	b.Add(7)
	if !b.Contains(7) {
		t.Fatal("expected 7 after duplicate Add")
	}
}

func TestRoaring_ArrayToBitmapPromotion(t *testing.T) {
	b := roaring.New()
	// Force array container past maxArraySize (4096) within one high-16 bucket.
	const n = 4100
	for i := uint32(0); i < n; i++ {
		b.Add(i)
	}
	for i := uint32(0); i < n; i += 137 {
		if !b.Contains(i) {
			t.Fatalf("missing %d after promotion", i)
		}
	}
	if !b.Contains(n-1) {
		t.Fatal("missing last value after promotion")
	}
}
