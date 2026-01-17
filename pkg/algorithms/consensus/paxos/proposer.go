package paxos

import (
	"errors"
	"sync"
)

// Proposal represents a value being proposed.
type Proposal struct {
	ID    int // Sequence Number
	Value interface{}
}

// Proposer initiates a Paxos round.
type Proposer struct {
	id         int
	numPeers   int
	transport  Transport
	proposalID int
	mu         sync.Mutex
}

// Acceptor accepts proposals.
type Acceptor struct {
	lastPromisedID int
	acceptedID     int
	acceptedValue  interface{}
	mu             sync.Mutex
}

// Learner learns the agreed value.
type Learner struct {
	acceptedCount map[int]int
	mu            sync.Mutex
}

// Transport abstracts network.
type Transport interface {
	Prepare(peerID int, proposalID int) (promised bool, acceptedID int, acceptedValue interface{})
	Accept(peerID int, proposalID int, value interface{}) (accepted bool)
}

func NewProposer(id int, numPeers int, transport Transport) *Proposer {
	return &Proposer{
		id:        id,
		numPeers:  numPeers,
		transport: transport,
	}
}

func NewAcceptor() *Acceptor {
	return &Acceptor{
		lastPromisedID: -1,
		acceptedID:     -1,
	}
}

// Propose attempts to reach consensus on a value.
func (p *Proposer) Propose(value interface{}) (bool, error) {
	p.mu.Lock()
	p.proposalID++ // Simple increment, real Paxos needs unique IDs (e.g., timestamp + nodeID)
	propID := p.proposalID
	p.mu.Unlock()

	// Phase 1: Prepare
	promises := 0
	highestAcceptedID := -1
	var highestValue interface{}

	for i := 0; i < p.numPeers; i++ {
		// Assume we call ourselves too via transport
		promised, accID, accVal := p.transport.Prepare(i, propID)
		if promised {
			promises++
			if accID > highestAcceptedID {
				highestAcceptedID = accID
				highestValue = accVal
			}
		}
	}

	if promises <= p.numPeers/2 {
		return false, errors.New("majority promises not received")
	}

	// Use highest seen value if any, otherwise propose ours
	valToPropose := value
	if highestValue != nil {
		valToPropose = highestValue
	}

	// Phase 2: Accept
	accepts := 0
	for i := 0; i < p.numPeers; i++ {
		if p.transport.Accept(i, propID, valToPropose) {
			accepts++
		}
	}

	if accepts <= p.numPeers/2 {
		return false, errors.New("majority accepts not received")
	}

	return true, nil
}

// ReceivePrepare handles a Prepare message (Acceptor logic).
func (a *Acceptor) ReceivePrepare(proposalID int) (bool, int, interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if proposalID > a.lastPromisedID {
		a.lastPromisedID = proposalID
		return true, a.acceptedID, a.acceptedValue
	}
	return false, -1, nil
}

// ReceiveAccept handles an Accept message (Acceptor logic).
func (a *Acceptor) ReceiveAccept(proposalID int, value interface{}) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if proposalID >= a.lastPromisedID {
		a.lastPromisedID = proposalID
		a.acceptedID = proposalID
		a.acceptedValue = value
		return true
	}
	return false
}
