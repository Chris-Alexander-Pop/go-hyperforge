package deque_test

import (
	"testing"

	"github.com/chris-alexander-pop/system-design-library/pkg/datastructures/deque"
)

func TestDeque_Regression_NonPowerOfTwo(t *testing.T) {
	// This test verifies that providing a non-power-of-two capacity
	// does not break the circular buffer logic (which relies on bitwise AND).
	// The implementation should internally round up the capacity to the next power of 2.

	// Create with capacity 10 (not power of 2)
	d := deque.New[int](10)

	// Push 1 at back.
	d.PushBack(1)
	// Push 2 at front.
	d.PushFront(2)

	// Expected order: 2, 1.

	val1, ok1 := d.PopFront()
	if !ok1 || val1 != 2 {
		t.Fatalf("First pop: Expected 2, got %v", val1)
	}

	val2, ok2 := d.PopFront()
	if !ok2 || val2 != 1 {
		t.Fatalf("Second pop: Expected 1, got %v", val2)
	}
}
