package merkle

import (
	"testing"
)

func TestMerkleTree(t *testing.T) {
	data := [][]byte{
		[]byte("block1"),
		[]byte("block2"),
		[]byte("block3"),
		[]byte("block4"),
	}

	tree, err := New(data)
	if err != nil {
		t.Fatalf("Failed to create tree: %v", err)
	}

	t.Run("RootHash", func(t *testing.T) {
		if tree.Root == nil {
			t.Fatal("Root should not be nil")
		}
		if len(tree.Root.Hash) != 32 {
			t.Errorf("Expected 32-byte hash, got %d", len(tree.Root.Hash))
		}
	})

	t.Run("ProofVerification", func(t *testing.T) {
		// Verify proof for block 2 (index 1)
		idx := 1
		proof, err := tree.GetProof(idx)
		if err != nil {
			t.Fatalf("Failed to get proof: %v", err)
		}

		valid := VerifyProof(tree.Root.Hash, data[idx], proof, idx)
		if !valid {
			t.Error("Proof verification failed for valid block")
		}
	})

	t.Run("InvalidProof", func(t *testing.T) {
		idx := 0
		proof, _ := tree.GetProof(idx)

		// Tamper with data
		fakeData := []byte("tampered")
		valid := VerifyProof(tree.Root.Hash, fakeData, proof, idx)
		if valid {
			t.Error("Proof verification should fail for tampered data")
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		_, err := New([][]byte{})
		if err == nil {
			t.Error("Expected error for empty data")
		}
	})

	t.Run("OddNumberOfLeaves", func(t *testing.T) {
		oddData := [][]byte{
			[]byte("a"),
			[]byte("b"),
			[]byte("c"),
		}
		tree, _ := New(oddData)

		// Should verify last element correctly
		proof, _ := tree.GetProof(2)
		if !VerifyProof(tree.Root.Hash, oddData[2], proof, 2) {
			t.Error("Failed to verify odd leaf")
		}
	})
}
