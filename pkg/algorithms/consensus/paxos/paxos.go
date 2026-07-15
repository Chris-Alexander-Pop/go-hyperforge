package paxos

import (
	"sync"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Educational sketch only — see package doc. Not a production consensus library.

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

// slotState is per-decree acceptor memory.
type slotState struct {
	lastPromisedID int
	acceptedID     int
	AcceptedValue  interface{}
}

// Acceptor accepts proposals per slot (instance).
type Acceptor struct {
	mu    sync.Mutex
	slots map[int]*slotState
}

// AcceptedValue returns the accepted value for slot 0 (single-decree convenience).
func (a *Acceptor) AcceptedValue() interface{} {
	return a.AcceptedValueAt(0)
}

// AcceptedValueAt returns the accepted value for a slot, if any.
func (a *Acceptor) AcceptedValueAt(slot int) interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.slots[slot]
	if s == nil {
		return nil
	}
	return s.AcceptedValue
}

// Learner collects Accept notifications per slot and learns a value once a
// majority of acceptors report the same (proposalID, value) for that slot.
type Learner struct {
	numPeers int
	mu       sync.Mutex
	// slot -> proposalID -> value -> count
	counts  map[int]map[int]map[interface{}]int
	learned map[int]interface{}
}

// Transport abstracts network. Slot distinguishes Multi-Paxos instances.
type Transport interface {
	Prepare(peerID int, slot, proposalID int) (promised bool, acceptedID int, AcceptedValue interface{})
	Accept(peerID int, slot, proposalID int, value interface{}) (accepted bool)
}

// LearnTransport optionally delivers Accept outcomes to a Learner for a slot.
type LearnTransport interface {
	Transport
	NotifyLearn(slot, proposalID int, value interface{})
}

func NewProposer(id int, numPeers int, transport Transport) *Proposer {
	return &Proposer{
		id:        id,
		numPeers:  numPeers,
		transport: transport,
	}
}

func NewAcceptor() *Acceptor {
	return &Acceptor{slots: make(map[int]*slotState)}
}

func (a *Acceptor) state(slot int) *slotState {
	s, ok := a.slots[slot]
	if !ok {
		s = &slotState{lastPromisedID: -1, acceptedID: -1}
		a.slots[slot] = s
	}
	return s
}

// NewLearner creates a Learner that learns after majority accepts per slot.
func NewLearner(numPeers int) *Learner {
	return &Learner{
		numPeers: numPeers,
		counts:   make(map[int]map[int]map[interface{}]int),
		learned:  make(map[int]interface{}),
	}
}

// Observe records an Accept for slot/proposalID/value. Returns true when a
// majority has accepted the same pair for that slot.
func (l *Learner) Observe(slot, proposalID int, value interface{}) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.learned[slot]; ok {
		return true
	}
	byProp, ok := l.counts[slot]
	if !ok {
		byProp = make(map[int]map[interface{}]int)
		l.counts[slot] = byProp
	}
	byVal, ok := byProp[proposalID]
	if !ok {
		byVal = make(map[interface{}]int)
		byProp[proposalID] = byVal
	}
	byVal[value]++
	if byVal[value] > l.numPeers/2 {
		l.learned[slot] = value
		return true
	}
	return false
}

// Value returns the learned value for a slot, if any.
func (l *Learner) Value(slot int) (interface{}, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.learned[slot]
	return v, ok
}

// Propose attempts to reach consensus on a value (single decree, slot 0).
func (p *Proposer) Propose(value interface{}) (bool, error) {
	ok, _, err := p.propose(0, value)
	return ok, err
}

func (p *Proposer) propose(slot int, value interface{}) (bool, int, error) {
	p.mu.Lock()
	p.proposalID++ // Simple increment; real Paxos needs unique IDs (e.g. timestamp + nodeID)
	propID := p.proposalID
	p.mu.Unlock()

	// Phase 1: Prepare
	promises := 0
	highestAcceptedID := -1
	var highestValue interface{}

	for i := 0; i < p.numPeers; i++ {
		promised, accID, accVal := p.transport.Prepare(i, slot, propID)
		if promised {
			promises++
			if accID > highestAcceptedID {
				highestAcceptedID = accID
				highestValue = accVal
			}
		}
	}

	if promises <= p.numPeers/2 {
		return false, propID, errors.FailedPrecondition("majority promises not received", nil)
	}

	valToPropose := value
	if highestValue != nil {
		valToPropose = highestValue
	}

	// Phase 2: Accept
	accepts := 0
	for i := 0; i < p.numPeers; i++ {
		if p.transport.Accept(i, slot, propID, valToPropose) {
			accepts++
			if lt, ok := p.transport.(LearnTransport); ok {
				lt.NotifyLearn(slot, propID, valToPropose)
			}
		}
	}

	if accepts <= p.numPeers/2 {
		return false, propID, errors.FailedPrecondition("majority accepts not received", nil)
	}

	return true, propID, nil
}

// ReceivePrepare handles a Prepare message (Acceptor logic) for a slot.
func (a *Acceptor) ReceivePrepare(slot, proposalID int) (bool, int, interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := a.state(slot)
	if proposalID > s.lastPromisedID {
		s.lastPromisedID = proposalID
		return true, s.acceptedID, s.AcceptedValue
	}
	return false, -1, nil
}

// ReceiveAccept handles an Accept message (Acceptor logic) for a slot.
func (a *Acceptor) ReceiveAccept(slot, proposalID int, value interface{}) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	s := a.state(slot)
	if proposalID >= s.lastPromisedID {
		s.lastPromisedID = proposalID
		s.acceptedID = proposalID
		s.AcceptedValue = value
		return true
	}
	return false
}

// MultiPaxos is an educational Multi-Paxos sketch: successive single-decree
// rounds with monotonic slots. There is no leader election, reconfiguration,
// or durable log — only in-process sequencing for learning/API exploration.
type MultiPaxos struct {
	proposer *Proposer
	learner  *Learner
	mu       sync.Mutex
	nextSlot int
	// Log maps slot -> chosen value.
	Log map[int]interface{}
}

// NewMultiPaxos wires a Proposer and Learner for sequential decrees.
func NewMultiPaxos(proposerID, numPeers int, transport Transport, learner *Learner) *MultiPaxos {
	if learner == nil {
		learner = NewLearner(numPeers)
	}
	return &MultiPaxos{
		proposer: NewProposer(proposerID, numPeers, transport),
		learner:  learner,
		Log:      make(map[int]interface{}),
	}
}

// ProposeSlot runs one decree and, on success, records the value at the next slot.
func (m *MultiPaxos) ProposeSlot(value interface{}) (slot int, ok bool, err error) {
	m.mu.Lock()
	slot = m.nextSlot
	m.mu.Unlock()

	ok, _, err = m.proposer.propose(slot, value)
	if err != nil || !ok {
		return slot, false, err
	}

	learned := value
	if m.learner != nil {
		if v, ready := m.learner.Value(slot); ready {
			learned = v
		}
	}

	m.mu.Lock()
	m.Log[slot] = learned
	m.nextSlot++
	m.mu.Unlock()
	return slot, true, nil
}

// Chosen returns the value for a slot, if present.
func (m *MultiPaxos) Chosen(slot int) (interface{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.Log[slot]
	return v, ok
}
