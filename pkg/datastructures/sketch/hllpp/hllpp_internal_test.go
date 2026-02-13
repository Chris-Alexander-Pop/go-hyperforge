package hllpp

import (
	"hash/fnv"
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

func TestHashCorrectness(t *testing.T) {
	inputs := []string{"test", "hello", "world", "1", "2", "3", "", "a very long string to test hashing properly"}

	for _, s := range inputs {
		data := []byte(s)

		// Standard FNV
		f := fnv.New64a()
		f.Write(data)
		expected := f.Sum64()

		// My implementation
		actual := hash64(data)

		if actual != expected {
			t.Errorf("Hash mismatch for %q: expected %x, got %x", s, expected, actual)
		}
	}
}
