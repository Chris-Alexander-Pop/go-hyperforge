package skiplist_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/skiplist"
	"testing"
)

func TestSkipList(t *testing.T) {
	sl := skiplist.New[int, string]()

	sl.Set(10, "foo")
	sl.Set(20, "bar")

	if val, ok := sl.Get(10); !ok || val != "foo" {
		t.Errorf("Expected foo, got %v", val)
	}

	if _, ok := sl.Get(99); ok {
		t.Error("Expected key 99 to be missing")
	}

	sl.Delete(10)
	if _, ok := sl.Get(10); ok {
		t.Error("Expected key 10 to be deleted")
	}
}
