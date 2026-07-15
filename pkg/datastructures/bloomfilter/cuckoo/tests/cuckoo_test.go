package cuckoo_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/bloomfilter/cuckoo"
)

func TestCuckoo_AddContainsDelete(t *testing.T) {
	f := cuckoo.New(128)
	item := []byte("hello")

	if f.Contains(item) {
		t.Fatal("empty filter should not contain item")
	}
	if !f.Add(item) {
		t.Fatal("Add should succeed")
	}
	if !f.Contains(item) {
		t.Fatal("expected Contains after Add")
	}
	if !f.Delete(item) {
		t.Fatal("Delete should succeed")
	}
	if f.Contains(item) {
		t.Fatal("item should be gone after Delete")
	}
	if f.Delete(item) {
		t.Fatal("second Delete should fail")
	}
}

func TestCuckoo_MultipleItems(t *testing.T) {
	f := cuckoo.New(256)
	items := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	for _, it := range items {
		if !f.Add(it) {
			t.Fatalf("Add(%s) failed", it)
		}
	}
	for _, it := range items {
		if !f.Contains(it) {
			t.Fatalf("Contains(%s) false", it)
		}
	}
	if f.Contains([]byte("missing")) {
		t.Fatal("unexpected false positive for missing")
	}
}

func TestCuckoo_DuplicateAdd(t *testing.T) {
	f := cuckoo.New(64)
	item := []byte("dup")
	if !f.Add(item) || !f.Add(item) {
		t.Fatal("duplicate Add should succeed")
	}
}
