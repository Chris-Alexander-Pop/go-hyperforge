package lfu_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/lfu"
	"testing"
)

func TestLFU(t *testing.T) {
	c := lfu.New[string, int](2)

	c.Set("one", 1)
	c.Set("one", 1) // Frequency 2
	c.Set("two", 2) // Frequency 1

	c.Set("three", 3) // Should evict "two" (lowest frequency)

	if _, ok := c.Get("two"); ok {
		t.Error("Expected two to be evicted")
	}
	if _, ok := c.Get("one"); !ok {
		t.Error("Expected one to be present")
	}
}
