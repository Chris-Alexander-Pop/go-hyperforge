package hllpp

import (
	"fmt"
	"testing"
)

func TestMergeSparsePrecision(t *testing.T) {
	// Create HLLPP with p=4 (m=16 registers)
	h := New(4)

	// We want to test that the full 64-bit hash is preserved.
	// We will manually insert a hash into the sparse map that would be corrupted
	// by the previous implementation.

	// Original Hash: 0xA000000000000001
	// Top 4 bits: 0xA (1010 -> 10). Index should be 10.
	// Bottom 32 bits: 0x00000001.

	// Previous implementation:
	// 1. stored uint32(hash) -> 0x00000001
	// 2. mergeSparse: restored = uint64(0x00000001) << 32 -> 0x0000000100000000
	// 3. New Index: Top 4 bits of restored -> 0x0 (0000 -> 0). Index 0.

	// So if the fix is working, register 10 should be non-zero, and register 0 should be 0
	// (assuming no other items).

	originalHash := uint64(0xA000000000000001)
	h.sparse[originalHash] = struct{}{}

	// Force merge
	h.mergeSparse()

	// Manually set IsSparse to false as mergeSparse doesn't do it itself (Add does it)
	// Wait, check mergeSparse implementation.
	// mergeSparse clears the map but doesn't toggle the boolean. Add toggles it.
	// But mergeSparse is called by Add.
	// In this test we call mergeSparse manually. We should update IsSparse to match reality if we care,
	// but for checking registers it doesn't matter.

	// Check Register 10
	idx := originalHash >> (64 - h.p) // 10
	if h.registers[idx] == 0 {
		t.Errorf("Expected register %d to be updated, but it was 0", idx)
	}

	// Check Register 0 (where it would have gone in the buggy version)
	// In the buggy version, 0x1 became 0x100000000... which has index 0.
	// With the fix, register 0 should stay 0.
	if h.registers[0] != 0 {
		t.Errorf("Expected register 0 to be 0, but got %d. This implies the old buggy logic might still be active or a collision occurred.", h.registers[0])
	}
}

func TestAddString(t *testing.T) {
	h1 := New(10) // m=1024 registers
	h2 := New(10)

	// Add enough items to trigger dense mode (threshold is 256 for p=10)
	for i := 0; i < 1000; i++ {
		s := fmt.Sprintf("test-item-%d", i)
		h1.Add([]byte(s))
		h2.AddString(s)
	}

	if h1.Count() != h2.Count() {
		t.Errorf("Count mismatch: Add=%d, AddString=%d", h1.Count(), h2.Count())
	}

	// Verify dense mode
	if h1.IsSparse {
		t.Errorf("h1 should be in dense mode")
	}
	if h2.IsSparse {
		t.Errorf("h2 should be in dense mode")
	}

	// Verify registers match exactly
	for i := range h1.registers {
		if h1.registers[i] != h2.registers[i] {
			t.Errorf("Register %d mismatch: %d vs %d", i, h1.registers[i], h2.registers[i])
		}
	}
}
