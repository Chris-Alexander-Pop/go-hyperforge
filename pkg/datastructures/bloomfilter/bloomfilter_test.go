package bloomfilter

import (
	"encoding/binary"
	"hash/fnv"
	"testing"
)

func TestDoubleHashCorrectness(t *testing.T) {
	testCases := []string{
		"",
		"a",
		"ab",
		"abc",
		"hello world",
		"The quick brown fox jumps over the lazy dog",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
	}

	for _, tc := range testCases {
		data := []byte(tc)

		// Expected (using standard library)
		h := fnv.New128a()
		h.Write(data)
		sum := h.Sum(nil)

		expectedH1 := binary.BigEndian.Uint64(sum[0:8])
		expectedH2 := binary.BigEndian.Uint64(sum[8:16])

		// Actual (using internal function)
		actualH1, actualH2 := doubleHash(data)

		if expectedH1 != actualH1 || expectedH2 != actualH2 {
			t.Errorf("Mismatch for %q:\nExpected: %x %x\nActual:   %x %x", tc, expectedH1, expectedH2, actualH1, actualH2)
		}
	}
}
