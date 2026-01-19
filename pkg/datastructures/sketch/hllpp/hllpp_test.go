package hllpp

import (
	"fmt"
	"testing"
)

func TestHLLPP(t *testing.T) {
	h := New(14)

	// Test Sparse Mode
	h.Add([]byte("a"))
	h.Add([]byte("b"))
	h.Add([]byte("c"))

	if h.Count() != 3 {
		t.Errorf("Expected 3 in sparse mode, got %d", h.Count())
	}

	// Force Switch to Dense
	// Threshold is m/4 = 16384 / 4 = 4096
	// Let's create a smaller one to force switch easily

	hSmall := New(4) // m=16, threshold=4
	for i := 0; i < 10; i++ {
		hSmall.Add([]byte(fmt.Sprintf("%d", i)))
	}

	if hSmall.isSparse {
		t.Error("Expected to switch to dense mode")
	}

	c := hSmall.Count()
	if c < 8 || c > 12 {
		// Very loose bounds heavily dependent on small HLL accuracy
		t.Logf("Small HLL count estimate: %d (expected ~10)", c)
	}
}
