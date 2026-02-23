package hyperloglog

import (
	"hash/fnv"
	"testing"
)

func TestHashCorrectness(t *testing.T) {
	inputs := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("longer string to test hashing"),
		[]byte(""),
		[]byte{0, 1, 2, 3, 255},
	}

	for _, input := range inputs {
		// Standard library implementation + mix
		h := fnv.New64a()
		h.Write(input)
		expected := mix(h.Sum64())

		// Our implementation in hll.go
		actual := hashBytes(input)

		if actual != expected {
			t.Errorf("Hash mismatch for input %q: expected %v, got %v", input, expected, actual)
		}
	}
}

func TestStringHashCorrectness(t *testing.T) {
	inputs := []string{
		"hello",
		"world",
		"longer string to test hashing",
		"",
	}

	for _, input := range inputs {
		// Standard library implementation + mix
		h := fnv.New64a()
		h.Write([]byte(input))
		expected := mix(h.Sum64())

		// Our implementation in hll.go
		actual := hashString(input)

		if actual != expected {
			t.Errorf("Hash mismatch for input %q: expected %v, got %v", input, expected, actual)
		}
	}
}
