package raft

import (
	"math/rand"
	"sync"
	"time"
)

// State represents the Raft node state.
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

// LogEntry is a single log entry.
type LogEntry struct {
	Term    int
	Command interface{}
}

// Node represents a Raft node.
type Node struct {
	id          string
	peers       []string
	state       State
	currentTerm int
	votedFor    string
	log         []LogEntry

	commitIndex int

	// Leader state
	nextIndex  map[string]int
	matchIndex map[string]int

	mu sync.Mutex

	// Mock transport
	transport Transport

	// Channels
	stopCh  chan struct{}
	applyCh chan interface{}
}

// Transport abstracts RPCs.
type Transport interface {
	RequestVote(peer string, term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool)
	AppendEntries(peer string, term int, leaderID string, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int) (int, bool)
}

func New(id string, peers []string, transport Transport, applyCh chan interface{}) *Node {
	return &Node{
		id:         id,
		peers:      peers,
		state:      Follower,
		log:        make([]LogEntry, 0), // Log is 1-indexed usually, but 0 here for simplicity
		transport:  transport,
		stopCh:     make(chan struct{}),
		applyCh:    applyCh,
		nextIndex:  make(map[string]int),
		matchIndex: make(map[string]int),
	}
}

func (n *Node) Start() {
	go n.run()
}

func (n *Node) run() {
	for {
		switch n.state {
		case Follower:
			n.runFollower()
		case Candidate:
			n.runCandidate()
		case Leader:
			n.runLeader()
		}
	}
}

func (n *Node) runFollower() {
	timeout := randomTimeout()
	timer := time.NewTimer(timeout)

	// In real impl, we listen for RPCs here.
	// Since this is just the statemachine logic, we simulate waiting.
	// RPC handlers would reset this timer.

	select {
	case <-timer.C:
		n.mu.Lock()
		n.state = Candidate
		n.mu.Unlock()
	case <-n.stopCh:
		return
	}
}

func (n *Node) runCandidate() {
	n.mu.Lock()
	n.currentTerm++
	n.votedFor = n.id
	term := n.currentTerm
	// log info
	lastLogIndex := len(n.log) - 1
	lastLogTerm := 0
	if lastLogIndex >= 0 {
		lastLogTerm = n.log[lastLogIndex].Term
	}
	n.mu.Unlock()

	// Channel to signal state change (Leader or Follower) to wake up the loop
	notifyCh := make(chan struct{}, 1)

	// Request Votes
	votes := 1
	for _, peer := range n.peers {
		go func(p string) {
			t, granted := n.transport.RequestVote(p, term, n.id, lastLogIndex, lastLogTerm)
			if granted {
				n.mu.Lock()
				votes++
				if n.state == Candidate && votes > (len(n.peers)+1)/2 {
					n.state = Leader
					// Init leader state
					for _, p := range n.peers {
						n.nextIndex[p] = len(n.log)
						n.matchIndex[p] = -1
					}
					// Notify state change
					select {
					case notifyCh <- struct{}{}:
					default:
					}
				}
				n.mu.Unlock()
			} else if t > term {
				n.mu.Lock()
				n.state = Follower
				n.currentTerm = t
				n.votedFor = ""
				// Notify state change
				select {
				case notifyCh <- struct{}{}:
				default:
				}
				n.mu.Unlock()
			}
		}(peer)
	}

	timeout := randomTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-n.stopCh:
		return
	case <-notifyCh:
		return
	case <-timer.C:
		return
	}
}

func (n *Node) runLeader() {
	ticker := time.NewTicker(50 * time.Millisecond) // Heartbeat interval
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.mu.Lock()
			if n.state != Leader {
				n.mu.Unlock()
				return
			}
			n.mu.Unlock()
			n.sendHeartbeats()
		}
	}
}

func (n *Node) sendHeartbeats() {
	n.mu.Lock()
	term := n.currentTerm
	leaderID := n.id
	commit := n.commitIndex
	n.mu.Unlock()

	for _, peer := range n.peers {
		// Mock empty append entries
		go func(p string) {
			t, _ := n.transport.AppendEntries(p, term, leaderID, 0, 0, nil, commit)
			if t > term {
				n.mu.Lock()
				n.state = Follower
				n.currentTerm = t
				n.votedFor = ""
				n.mu.Unlock()
			}
		}(peer)
	}
}

func randomTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
