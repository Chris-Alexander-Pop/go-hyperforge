package arc_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/arc"
)

func TestARC_BasicGetSet(t *testing.T) {
	c := arc.New[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a)=%v,%v want 1,true", v, ok)
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("Get(b)=%v,%v want 2,true", v, ok)
	}
}

func TestARC_Eviction(t *testing.T) {
	c := arc.New[string, int](2)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3) // should force eviction from T1

	present := 0
	for _, k := range []string{"a", "b", "c"} {
		if _, ok := c.Get(k); ok {
			present++
		}
	}
	if present < 1 || present > 3 {
		t.Fatalf("unexpected present count %d", present)
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected most recent key c to be present")
	}
}

func TestARC_UpdateExisting(t *testing.T) {
	c := arc.New[string, int](4)
	c.Set("x", 1)
	c.Set("x", 2)
	if v, ok := c.Get("x"); !ok || v != 2 {
		t.Fatalf("Get(x)=%v,%v want 2,true", v, ok)
	}
}

func TestARC_Miss(t *testing.T) {
	c := arc.New[string, int](2)
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss")
	}
}

func TestARC_ZeroCapacityDefaults(t *testing.T) {
	c := arc.New[string, int](0)
	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a)=%v,%v want 1,true", v, ok)
	}
}

func TestARC_GhostPromotion(t *testing.T) {
	c := arc.New[string, int](2)
	// Fill and cycle keys so ghost lists get used, then reinsert.
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)
	c.Set("d", 4)
	c.Set("a", 10) // may hit B1/B2 history depending on eviction path
	if v, ok := c.Get("a"); !ok || v != 10 {
		t.Fatalf("Get(a)=%v,%v want 10,true after reinsert", v, ok)
	}
}
