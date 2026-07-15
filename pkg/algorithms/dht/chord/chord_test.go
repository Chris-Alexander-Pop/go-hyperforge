package chord_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/dht/chord"
)

func TestJoinStabilizeNotify(t *testing.T) {
	tr := chord.NewInProcessTransport()

	a := chord.New("node-a", tr)
	b := chord.New("node-b", tr)
	c := chord.New("node-c", tr)
	tr.Register(a)
	tr.Register(b)
	tr.Register(c)

	a.Create()
	if err := b.Join(a.Addr()); err != nil {
		t.Fatalf("b.Join: %v", err)
	}
	if err := c.Join(a.Addr()); err != nil {
		t.Fatalf("c.Join: %v", err)
	}

	// Several stabilize rounds so predecessors/successors converge.
	for i := 0; i < 6; i++ {
		for _, n := range []*chord.Node{a, b, c} {
			if err := n.Stabilize(); err != nil {
				t.Fatalf("Stabilize: %v", err)
			}
		}
	}

	for _, n := range []*chord.Node{a, b, c} {
		if n.Successor() == nil {
			t.Fatalf("%s has nil successor", n.Addr())
		}
		if n.Predecessor() == nil {
			t.Fatalf("%s has nil predecessor after stabilize", n.Addr())
		}
	}

	// Each node's successor's predecessor should be that node (ring consistency).
	for _, n := range []*chord.Node{a, b, c} {
		succ := n.Successor()
		var peer *chord.Node
		switch succ.Addr {
		case a.Addr():
			peer = a
		case b.Addr():
			peer = b
		case c.Addr():
			peer = c
		default:
			t.Fatalf("unknown successor %s", succ.Addr)
		}
		pred := peer.Predecessor()
		if pred == nil || pred.Addr != n.Addr() {
			t.Fatalf("%s: successor %s predecessor=%v want %s", n.Addr(), succ.Addr, pred, n.Addr())
		}
	}

	// FindSuccessor for each node ID should return that node.
	for _, n := range []*chord.Node{a, b, c} {
		got, err := a.FindSuccessor(n.ID())
		if err != nil {
			t.Fatalf("FindSuccessor(%s): %v", n.Addr(), err)
		}
		if got.Addr != n.Addr() {
			t.Fatalf("FindSuccessor(%s)=%s", n.Addr(), got.Addr)
		}
	}
}
