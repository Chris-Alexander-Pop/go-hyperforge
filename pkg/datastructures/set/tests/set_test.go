package set_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/set"
	"testing"
)

func TestSet(t *testing.T) {
	s := set.New[string]()

	s.Add("a")
	s.Add("b")

	if !s.Contains("a") {
		t.Error("Expected set to contain 'a'")
	}
	if s.Contains("c") {
		t.Error("Expected set to not contain 'c'")
	}

	s.Remove("a")
	if s.Contains("a") {
		t.Error("freed item should not be present")
	}
}
