package queue_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/queue"
	"testing"
)

func TestQueue(t *testing.T) {
	q := queue.New[int]()

	q.Enqueue(1)
	q.Enqueue(2)

	if q.Len() != 2 {
		t.Errorf("Expected length 2, got %d", q.Len())
	}

	if val, ok := q.Dequeue(); !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}
	if val, ok := q.Dequeue(); !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}
}
