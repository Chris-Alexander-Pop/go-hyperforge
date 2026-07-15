package chord

import (
	"crypto/sha1"
	"math/big"
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Educational sketch only — see package doc. Not a production DHT.

const (
	m = 160 // Key size in bits (SHA-1)
)

// Node represents a node in the Chord ring.
type Node struct {
	id          *big.Int
	addr        string
	successor   *RemoteNode
	predecessor *RemoteNode
	finger      []*RemoteNode
	transport   Transport
	mu          sync.RWMutex
}

// RemoteNode is a remote Chord peer reference.
type RemoteNode struct {
	ID   *big.Int
	Addr string
}

// Transport abstracts Chord RPCs. InProcessTransport implements this for tests.
type Transport interface {
	FindSuccessor(addr string, id *big.Int) (*RemoteNode, error)
	GetPredecessor(addr string) (*RemoteNode, error)
	Notify(addr string, n *RemoteNode) error
}

// InProcessTransport routes Chord RPCs to local Node instances by address.
type InProcessTransport struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

// NewInProcessTransport creates an empty in-process transport registry.
func NewInProcessTransport() *InProcessTransport {
	return &InProcessTransport{nodes: make(map[string]*Node)}
}

// Register adds a node so peers can reach it by address.
func (t *InProcessTransport) Register(n *Node) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[n.addr] = n
}

func (t *InProcessTransport) node(addr string) (*Node, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	n, ok := t.nodes[addr]
	if !ok {
		return nil, errors.NotFound("chord: unknown peer "+addr, nil)
	}
	return n, nil
}

func (t *InProcessTransport) FindSuccessor(addr string, id *big.Int) (*RemoteNode, error) {
	n, err := t.node(addr)
	if err != nil {
		return nil, err
	}
	return n.FindSuccessor(id)
}

func (t *InProcessTransport) GetPredecessor(addr string) (*RemoteNode, error) {
	n, err := t.node(addr)
	if err != nil {
		return nil, err
	}
	return n.Predecessor(), nil
}

func (t *InProcessTransport) Notify(addr string, remote *RemoteNode) error {
	n, err := t.node(addr)
	if err != nil {
		return err
	}
	n.Notify(remote)
	return nil
}

// New creates a Chord node bound to transport (may be nil until Create/Join).
func New(addr string, transport Transport) *Node {
	h := sha1.New()
	h.Write([]byte(addr))
	id := new(big.Int).SetBytes(h.Sum(nil))

	return &Node{
		id:        id,
		addr:      addr,
		finger:    make([]*RemoteNode, m),
		transport: transport,
	}
}

// ID returns the node's Chord identifier.
func (n *Node) ID() *big.Int {
	return new(big.Int).Set(n.id)
}

// Addr returns the node's address.
func (n *Node) Addr() string { return n.addr }

// Self returns this node as a RemoteNode.
func (n *Node) Self() *RemoteNode {
	return &RemoteNode{ID: new(big.Int).Set(n.id), Addr: n.addr}
}

// Successor returns the current successor (may be nil before Create/Join).
func (n *Node) Successor() *RemoteNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return copyRemote(n.successor)
}

// Predecessor returns the current predecessor (may be nil).
func (n *Node) Predecessor() *RemoteNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return copyRemote(n.predecessor)
}

func copyRemote(r *RemoteNode) *RemoteNode {
	if r == nil {
		return nil
	}
	return &RemoteNode{ID: new(big.Int).Set(r.ID), Addr: r.Addr}
}

// Create starts a new ring with this node as its own successor.
func (n *Node) Create() {
	n.mu.Lock()
	defer n.mu.Unlock()
	self := &RemoteNode{ID: new(big.Int).Set(n.id), Addr: n.addr}
	n.successor = self
	n.predecessor = nil
	n.finger[0] = self
}

// Join joins an existing ring via bootstrapAddr using Transport RPCs.
func (n *Node) Join(bootstrapAddr string) error {
	if n.transport == nil {
		return errors.FailedPrecondition("chord: transport required for Join", nil)
	}
	succ, err := n.transport.FindSuccessor(bootstrapAddr, n.id)
	if err != nil {
		return err
	}
	n.mu.Lock()
	n.successor = succ
	n.predecessor = nil
	n.finger[0] = succ
	n.mu.Unlock()
	return nil
}

// FindSuccessor finds the successor node for a given ID.
func (n *Node) FindSuccessor(id *big.Int) (*RemoteNode, error) {
	n.mu.RLock()
	succ := n.successor
	if succ == nil {
		n.mu.RUnlock()
		return nil, errors.FailedPrecondition("chord: no successor", nil)
	}
	if between(n.id, succ.ID, id) || id.Cmp(succ.ID) == 0 {
		out := copyRemote(succ)
		n.mu.RUnlock()
		return out, nil
	}
	pred := n.closestPrecedingNodeLocked(id)
	localAddr := n.addr
	n.mu.RUnlock()

	if pred.Addr == localAddr || pred.ID.Cmp(n.id) == 0 {
		return copyRemote(succ), nil
	}
	if n.transport == nil {
		return nil, errors.FailedPrecondition("chord: transport required for remote FindSuccessor", nil)
	}
	return n.transport.FindSuccessor(pred.Addr, id)
}

func (n *Node) closestPrecedingNodeLocked(id *big.Int) *RemoteNode {
	for i := m - 1; i >= 0; i-- {
		fing := n.finger[i]
		if fing != nil && between(n.id, id, fing.ID) {
			return copyRemote(fing)
		}
	}
	return &RemoteNode{ID: new(big.Int).Set(n.id), Addr: n.addr}
}

// Stabilize verifies immediate successor and notifies it (Chord stabilize).
func (n *Node) Stabilize() error {
	n.mu.RLock()
	succ := copyRemote(n.successor)
	transport := n.transport
	self := &RemoteNode{ID: new(big.Int).Set(n.id), Addr: n.addr}
	n.mu.RUnlock()

	if succ == nil {
		return errors.FailedPrecondition("chord: no successor", nil)
	}
	if transport == nil {
		return errors.FailedPrecondition("chord: transport required for Stabilize", nil)
	}

	x, err := transport.GetPredecessor(succ.Addr)
	if err != nil {
		return err
	}
	if x != nil && between(n.id, succ.ID, x.ID) {
		n.mu.Lock()
		n.successor = x
		n.finger[0] = x
		succ = copyRemote(x)
		n.mu.Unlock()
	}
	return transport.Notify(succ.Addr, self)
}

// Notify may update this node's predecessor when n_ might be a better one.
func (n *Node) Notify(n_ *RemoteNode) {
	if n_ == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.predecessor == nil || between(n.predecessor.ID, n.id, n_.ID) {
		n.predecessor = copyRemote(n_)
	}
}

// between checks if key is in (n1, n2) on the Chord ring.
func between(n1, n2, key *big.Int) bool {
	if n1.Cmp(n2) < 0 {
		return key.Cmp(n1) > 0 && key.Cmp(n2) < 0
	}
	return key.Cmp(n1) > 0 || key.Cmp(n2) < 0
}
