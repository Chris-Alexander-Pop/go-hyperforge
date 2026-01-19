package paxos

import (
	"testing"
)

type MockTransport struct {
	peers map[int]*Acceptor
}

func (t *MockTransport) Prepare(peerID int, proposalID int) (promised bool, acceptedID int, acceptedValue interface{}) {
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
	acceptors := map[int]*Acceptor{
		0: NewAcceptor(),
		1: NewAcceptor(),
		2: NewAcceptor(),
	}

	transport := &MockTransport{peers: acceptors}

	// Proposer logic
	proposer := NewProposer(100, 3, transport)

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
		if a.acceptedValue == "ValueA" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("Majority consensus not reached, count=%d", count)
	}
}
