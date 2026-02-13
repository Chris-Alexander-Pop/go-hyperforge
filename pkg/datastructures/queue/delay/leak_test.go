package delay

import (
	"testing"
	"time"
)

type LargeStruct struct {
	data [1024]byte // Smaller struct for deterministic check
}

func TestMemoryLeakDirect(t *testing.T) {
	q := New[*LargeStruct]()

	item := &LargeStruct{}

	// Enqueue
	q.Enqueue(item, 0)

	// Ensure item is ready
	time.Sleep(1 * time.Millisecond)

	// Dequeue
	val, ok := q.Dequeue()
	if !ok {
		t.Fatal("expected item")
	}
	if val != item {
		t.Fatal("expected same item")
	}

	// Check the backing array
	// q.items is a slice. We want to see if the element beyond len is still set.
	// Since we popped, len should be 0.
	if len(q.items) != 0 {
		t.Fatalf("expected len 0, got %d", len(q.items))
	}

	// Access the underlying array by re-slicing up to capacity
	backing := q.items[:cap(q.items)]
	if len(backing) < 1 {
		t.Fatal("expected backing array to have capacity")
	}

	// With the fix, the popped item (at index 0) should be nil.
	if backing[0] != nil {
		t.Error("Memory leak detected: backing array still holds reference to popped item")
	}
}
