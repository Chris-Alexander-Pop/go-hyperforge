package linkedlist_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/linkedlist"
	"testing"
)

func TestLinkedList(t *testing.T) {
	l := linkedlist.New[int]()

	l.PushBack(1)
	l.PushBack(2)
	l.PushFront(0)

	if l.Len() != 3 {
		t.Errorf("Expected length 3, got %d", l.Len())
	}
}
