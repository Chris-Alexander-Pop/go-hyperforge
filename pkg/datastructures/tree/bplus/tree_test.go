package bplus

import "testing"

func TestBPlusTree(t *testing.T) {
	tree := New[int, string]()

	// Insert enough to trigger split (degree 4 -> max 7 keys)
	for i := 0; i < 20; i++ {
		tree.Insert(i, "val")
	}

	if val, found := tree.Search(5); !found || val != "val" {
		t.Error("Failed to find 5")
	}

	if _, found := tree.Search(99); found {
		t.Error("Found non-existent key")
	}

	// Overwrite
	tree.Insert(5, "updated")
	if val, _ := tree.Search(5); val != "updated" {
		t.Error("Update failed")
	}
}
