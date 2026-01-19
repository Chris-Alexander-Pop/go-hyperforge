package avl

import "testing"

func TestAVLTree(t *testing.T) {
	tree := New[int, string]()

	t.Run("InsertAndGet", func(t *testing.T) {
		tree.Put(10, "ten")
		tree.Put(20, "twenty")
		tree.Put(5, "five")

		if val, found := tree.Get(10); !found || val != "ten" {
			t.Errorf("Expected ten, got %v (%v)", val, found)
		}
		if val, found := tree.Get(5); !found || val != "five" {
			t.Errorf("Expected five, got %v (%v)", val, found)
		}
		if _, found := tree.Get(99); found {
			t.Error("Expected not found for 99")
		}
	})

	t.Run("Balance", func(t *testing.T) {
		// Trigger rotations
		// Right-Right case (Insert 1, 2, 3) -> Left Rotate
		tree2 := New[int, int]()
		tree2.Put(1, 1)
		tree2.Put(2, 2)
		tree2.Put(3, 3)

		// Root should be 2 if balanced
		if tree2.root.key != 2 {
			t.Errorf("Expected root 2 after rotation, got %v", tree2.root.key)
		}

		// Check height validity
		if height(tree2.root) != 2 {
			t.Errorf("Expected height 2, got %d", height(tree2.root))
		}
	})
}
