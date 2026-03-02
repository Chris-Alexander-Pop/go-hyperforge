package bloomfilter

import (
	"encoding/binary"
	"hash/fnv"
	"testing"
)

func TestDoubleHashCorrectness(t *testing.T) {
	tests := []string{
		"",
		"a",
		"ab",
		"abc",
		"hello world",
		"longer string to test hash function stability and correctness",
	}

	for _, s := range tests {
		data := []byte(s)

		// Standard library behavior check
		h := fnv.New128a()
		h.Write(data)
		expected := h.Sum(nil)

		// Our optimized implementation
		h0, h1 := doubleHash(data)

		// Convert inline result to bytes (big endian) to match fnv.Sum()
		got := make([]byte, 16)
		binary.BigEndian.PutUint64(got[0:8], h0)
		binary.BigEndian.PutUint64(got[8:16], h1)

		if string(got) != string(expected) {
			t.Errorf("Hash mismatch for %q:\nExpected: %x\nGot:      %x", s, expected, got)
		}
	}
}
