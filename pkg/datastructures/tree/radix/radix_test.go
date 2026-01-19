package radix

import "testing"

func TestRadixTree(t *testing.T) {
	tree := New[string]()

	t.Run("InsertAndGet", func(t *testing.T) {
		tree.Insert("apple", "fruit")
		tree.Insert("app", "app store")
		tree.Insert("ball", "game")

		if val, found := tree.Get("apple"); !found || val != "fruit" {
			t.Errorf("Expected fruit for apple, got %v", val)
		}
		if val, found := tree.Get("app"); !found || val != "app store" {
			t.Errorf("Expected app store for app, got %v", val)
		}
		if _, found := tree.Get("a"); found {
			t.Error("Expected not found for prefix 'a'")
		}
	})

	t.Run("EdgeSplitting", func(t *testing.T) {
		// Test complex splitting
		rt := New[int]()
		rt.Insert("water", 1)
		rt.Insert("wait", 2) // splits at 'wa' -> 'ter', 'it'

		if val, _ := rt.Get("water"); val != 1 {
			t.Error("Lost water")
		}
		if val, _ := rt.Get("wait"); val != 2 {
			t.Error("Lost wait")
		}

		rt.Insert("watch", 3) // splits 'it' if shared? No, splits 'wa' -> 't' -> 'ch', 'er', 'it' ??
		// 'wait' share 'wa' with 'water'.
		// 'wait' shares 'wa' with 'watch'.
		// The structure should handle it.

		if val, _ := rt.Get("watch"); val != 3 {
			t.Error("Lost watch")
		}
	})
}
