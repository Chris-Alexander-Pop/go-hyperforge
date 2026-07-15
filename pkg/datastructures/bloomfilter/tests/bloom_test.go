package bloomfilter_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/bloomfilter"
	"testing"
)

func TestBloomFilter(t *testing.T) {
	bf := bloomfilter.New(1000, 0.01)

	// Test basic operations
	bf.Add([]byte("apple"))
	bf.Add([]byte("banana"))

	if !bf.Contains([]byte("apple")) {
		t.Error("Expected apple to be present")
	}
	if !bf.Contains([]byte("banana")) {
		t.Error("Expected banana to be present")
	}
	if bf.Contains([]byte("cherry")) {
		// False positives are possible but unlikely with these params
		t.Log("False positive occurred for cherry")
	}
}
