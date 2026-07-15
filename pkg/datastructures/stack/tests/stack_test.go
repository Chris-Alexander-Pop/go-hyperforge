package stack_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/stack"
	"testing"
)

func TestStack(t *testing.T) {
	s := stack.New[int]()

	s.Push(1)
	s.Push(2)

	if s.Len() != 2 {
		t.Errorf("Expected length 2, got %d", s.Len())
	}

	if val, ok := s.Pop(); !ok || val != 2 {
		t.Errorf("Expected 2, got %v", val)
	}
	if val, ok := s.Peek(); !ok || val != 1 {
		t.Errorf("Expected 1, got %v", val)
	}
}
