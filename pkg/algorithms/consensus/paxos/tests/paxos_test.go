package paxos_test

import (
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/consensus/paxos"
)

type MockTransport struct {
	peers   map[int]*paxos.Acceptor
	learner *paxos.Learner
}

func (t *MockTransport) Prepare(peerID int, slot, proposalID int) (promised bool, acceptedID int, AcceptedValue interface{}) {
	if peer, ok := t.peers[peerID]; ok {
		return peer.ReceivePrepare(slot, proposalID)
	}
	return false, -1, nil
}

func (t *MockTransport) Accept(peerID int, slot, proposalID int, value interface{}) (accepted bool) {
	if peer, ok := t.peers[peerID]; ok {
		return peer.ReceiveAccept(slot, proposalID, value)
	}
	return false
}

func (t *MockTransport) NotifyLearn(slot, proposalID int, value interface{}) {
	if t.learner != nil {
		t.learner.Observe(slot, proposalID, value)
	}
}

func TestPaxosFlow(t *testing.T) {
	acceptors := map[int]*paxos.Acceptor{
		0: paxos.NewAcceptor(),
		1: paxos.NewAcceptor(),
		2: paxos.NewAcceptor(),
	}
	learner := paxos.NewLearner(3)
	transport := &MockTransport{peers: acceptors, learner: learner}
	proposer := paxos.NewProposer(100, 3, transport)

	success, err := proposer.Propose("ValueA")
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}
	if !success {
		t.Errorf("Propose returned false, expected true")
	}

	count := 0
	for _, a := range acceptors {
		if a.AcceptedValue() == "ValueA" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("Majority consensus not reached, count=%d", count)
	}

	got, ok := learner.Value(0)
	if !ok || got != "ValueA" {
		t.Fatalf("learner got=%v ok=%v", got, ok)
	}
}

func TestLearnerQuorum(t *testing.T) {
	l := paxos.NewLearner(3)
	// Majority of 3 is 2 (count > n/2 with integer division).
	if l.Observe(0, 1, "x") {
		t.Fatal("should not learn on first accept")
	}
	if !l.Observe(0, 1, "x") {
		t.Fatal("expected learn on second accept (majority)")
	}
	v, ok := l.Value(0)
	if !ok || v != "x" {
		t.Fatalf("got=%v ok=%v", v, ok)
	}
}

func TestMultiPaxosSlots(t *testing.T) {
	acceptors := map[int]*paxos.Acceptor{
		0: paxos.NewAcceptor(),
		1: paxos.NewAcceptor(),
		2: paxos.NewAcceptor(),
	}
	learner := paxos.NewLearner(3)
	transport := &MockTransport{peers: acceptors, learner: learner}
	mp := paxos.NewMultiPaxos(1, 3, transport, learner)

	for i, val := range []string{"a", "b", "c"} {
		slot, ok, err := mp.ProposeSlot(val)
		if err != nil || !ok {
			t.Fatalf("ProposeSlot(%q): slot=%d ok=%v err=%v", val, slot, ok, err)
		}
		if slot != i {
			t.Fatalf("slot=%d want %d", slot, i)
		}
		got, ok := mp.Chosen(slot)
		if !ok || got != val {
			t.Fatalf("Chosen(%d)=%v ok=%v want %q", slot, got, ok, val)
		}
	}
}
