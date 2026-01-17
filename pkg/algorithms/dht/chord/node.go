package chord

import (
	"crypto/sha1"
	"errors"
	"math/big"
	"sync"
)

const (
	m = 160 // Key size in bits (SHA-1)
)

// Node represents a node in the Chord ring.
type Node struct {
	id          *big.Int
	addr        string
	successor   *RemoteNode
	predecessor *RemoteNode
	finger      []*RemoteNode // Finger table
	mu          sync.RWMutex
}

type RemoteNode struct {
	ID   *big.Int
	Addr string
}

func New(addr string) *Node {
	h := sha1.New()
	h.Write([]byte(addr))
	id := new(big.Int).SetBytes(h.Sum(nil))

	return &Node{
		id:     id,
		addr:   addr,
		finger: make([]*RemoteNode, m),
	}
}

// FindSuccessor finds the successor node for a given ID.
func (n *Node) FindSuccessor(id *big.Int) (*RemoteNode, error) {
	n.mu.RLock()
	if between(n.id, n.successor.ID, id) || id.Cmp(n.successor.ID) == 0 {
		n.mu.RUnlock()
		return n.successor, nil
	}
	n.mu.RUnlock()

	// Forward to closest preceding node
	pred := n.closestPrecedingNode(id)
	if pred.ID.Cmp(n.id) == 0 {
		// Should not happen in stable ring unless single node
		return n.successor, nil
	}

	// In real impl, this is an RPC.
	// Here we simulate recursion/RPC by assuming we can just "Call" it.
	// return RPC(pred.Addr).FindSuccessor(id)
	return nil, errors.New("RPC not implemented")
}

func (n *Node) closestPrecedingNode(id *big.Int) *RemoteNode {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for i := m - 1; i >= 0; i-- {
		fing := n.finger[i]
		if fing != nil {
			if between(n.id, id, fing.ID) {
				return fing
			}
		}
	}
	return &RemoteNode{ID: n.id, Addr: n.addr}
}

// between checks if key is in (n1, n2)
func between(n1, n2, key *big.Int) bool {
	if n1.Cmp(n2) < 0 {
		return key.Cmp(n1) > 0 && key.Cmp(n2) < 0
	}
	return key.Cmp(n1) > 0 || key.Cmp(n2) < 0
}
