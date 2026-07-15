package paxos_test

import (
	"github.com/chris-alexander-pop/go-hyperforge/pkg/algorithms/consensus/paxos"
	"testing"
)

type MockTransport struct {
	peers map[int]*paxos.Acceptor
}

func (t *MockTransport) Prepare(peerID int, proposalID int) (promised bool, acceptedID int, AcceptedValue interface{}) {
	if peer, ok := t.peers[peerID]; ok {
		return peer.ReceivePrepare(proposalID)
	}
	return false, -1, nil
}

func (t *MockTransport) Accept(peerID int, proposalID int, value interface{}) (accepted bool) {
	if peer, ok := t.peers[peerID]; ok {
		return peer.ReceiveAccept(proposalID, value)
	}
	return false
}

func TestPaxosFlow(t *testing.T) {
	// Setup 3 acceptors
	acceptors := map[int]*paxos.Acceptor{
		0: paxos.NewAcceptor(),
		1: paxos.NewAcceptor(),
		2: paxos.NewAcceptor(),
	}

	transport := &MockTransport{peers: acceptors}

	// paxos.Proposer logic
	proposer := paxos.NewProposer(100, 3, transport)

	// Round 1: Propose "ValueA"
	success, err := proposer.Propose("ValueA")
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}
	if !success {
		t.Errorf("Propose returned false, expected true")
	}

	// Verify all acceptors have accepted matches
	count := 0
	for _, a := range acceptors {
		if a.AcceptedValue == "ValueA" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("Majority consensus not reached, count=%d", count)
	}
}
