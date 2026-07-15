package concurrentmap_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/concurrentmap"
	"strconv"
	"testing"
)

func TestConcurrentMap(t *testing.T) {
	m := concurrentmap.New[string, int](32)

	// Test Set/Get
	m.Set("foo", 123)
	val, ok := m.Get("foo")
	if !ok || val != 123 {
		t.Errorf("Expected 123, got %v", val)
	}

	// Test Remove
	m.Delete("foo")
	if _, ok := m.Get("foo"); ok {
		t.Error("freed item should not be present")
	}

	// Test Count
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	if m.Len() != 100 {
		t.Errorf("Expected 100 items, got %d", m.Len())
	}
}
