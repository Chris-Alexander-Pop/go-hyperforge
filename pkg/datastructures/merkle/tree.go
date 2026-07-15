package merkle

import (
	"crypto/sha256"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Tree represents a Merkle Tree.
type Tree struct {
	Root         *Node
	Leaves       []*Node
	hashStrategy func([]byte) []byte
}

// Node represents a node in the Merkle Tree.
type Node struct {
	Parent *Node
	Left   *Node
	Right  *Node
	Hash   []byte
	Data   []byte
	IsLeaf bool
}

// New creates a new Merkle Tree from the given data blocks.
func New(data [][]byte) (*Tree, error) {
	if len(data) == 0 {
		return nil, errors.InvalidArgument("cannot create tree with no data", nil)
	}

	leaves := make([]*Node, len(data))
	for i, d := range data {
		leaves[i] = &Node{
			Data:   d,
			Hash:   calculateHash(d),
			IsLeaf: true,
		}
	}

	root := buildTree(leaves)
	return &Tree{
		Root:         root,
		Leaves:       leaves,
		hashStrategy: calculateHash,
	}, nil
}

func calculateHash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func buildTree(nodes []*Node) *Node {
	if len(nodes) == 1 {
		return nodes[0]
	}

	var parents []*Node
	for i := 0; i < len(nodes); i += 2 {
		node1 := nodes[i]
		var node2 *Node

		if i+1 < len(nodes) {
			node2 = nodes[i+1]
		} else {
			// Duplicate odd node to balance
			node2 = &Node{
				Hash:   node1.Hash,
				Data:   node1.Data,
				IsLeaf: node1.IsLeaf, // In some impls, this is a distinct copy
			}
		}

		parentHash := append(node1.Hash, node2.Hash...)
		parent := &Node{
			Left:  node1,
			Right: node2,
			Hash:  calculateHash(parentHash),
		}
		node1.Parent = parent
		node2.Parent = parent
		parents = append(parents, parent)
	}

	return buildTree(parents)
}

// VerifyProof verifies a Merkle proof.
func VerifyProof(rootHash, data []byte, proof [][]byte, index int) bool {
	hash := calculateHash(data)

	for _, p := range proof {
		var siblingHash []byte
		if index%2 == 0 {
			siblingHash = append(hash, p...)
		} else {
			siblingHash = append(p, hash...)
		}
		hash = calculateHash(siblingHash)
		index /= 2
	}

	// Compare with root
	if len(hash) != len(rootHash) {
		return false
	}
	for i := range hash {
		if hash[i] != rootHash[i] {
			return false
		}
	}
	return true
}

// GetProof generates a Merkle proof for a leaf at the given index.
func (t *Tree) GetProof(index int) ([][]byte, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, errors.InvalidArgument("index out of bounds", nil)
	}

	var proof [][]byte
	current := t.Leaves[index]

	for current.Parent != nil {
		parent := current.Parent
		var sibling *Node
		if parent.Left == current {
			sibling = parent.Right
		} else {
			sibling = parent.Left
		}

		// Sibling should exist if built correctly (we duplicate for balance)
		if sibling != nil {
			proof = append(proof, sibling.Hash)
		} else {
			// Should not happen in this implementation
			proof = append(proof, current.Hash)
		}
		current = parent
	}

	return proof, nil
}
