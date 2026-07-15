package lru_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/lru"
	"testing"
)

func TestLRU(t *testing.T) {
	c := lru.New[string, int](2)

	c.Set("one", 1)
	c.Set("two", 2)

	if val, ok := c.Get("one"); !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}

	c.Set("three", 3) // Evicts "two" because "one" was just accessed

	if _, ok := c.Get("two"); ok {
		t.Error("Expected two to be evicted")
	}
	if _, ok := c.Get("three"); !ok {
		t.Error("Expected three to be present")
	}
}
