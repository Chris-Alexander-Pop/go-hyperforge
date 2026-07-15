package scalable_test

import (
	"fmt"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/datastructures/bloomfilter/scalable"
)

func TestScalable_AddContains(t *testing.T) {
	f := scalable.New(16, 0.01)
	item := []byte("alpha")
	if f.Contains(item) {
		t.Fatal("empty filter should not contain item")
	}
	f.Add(item)
	if !f.Contains(item) {
		t.Fatal("expected Contains after Add")
	}
	if f.Contains([]byte("beta")) {
		t.Fatal("unexpected Contains for unseen item")
	}
}

func TestScalable_GrowsAcrossLayers(t *testing.T) {
	f := scalable.New(8, 0.01)
	for i := 0; i < 40; i++ {
		f.Add([]byte(fmt.Sprintf("item-%d", i)))
	}
	for i := 0; i < 40; i++ {
		key := []byte(fmt.Sprintf("item-%d", i))
		if !f.Contains(key) {
			t.Fatalf("missing item-%d after growth", i)
		}
	}
}

func TestScalable_DuplicateAdd(t *testing.T) {
	f := scalable.New(8, 0.01)
	item := []byte("same")
	f.Add(item)
	f.Add(item)
	if !f.Contains(item) {
		t.Fatal("expected Contains after duplicate Add")
	}
}
