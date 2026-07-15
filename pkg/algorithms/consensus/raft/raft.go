package raft

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

// Educational sketch only — see package doc. Not a production Raft library.

// State represents the Raft node state.
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// ErrNotLeader is returned by Propose when this node is not the leader.
var ErrNotLeader = errors.New("raft: not leader")

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
	lastApplied int

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

// New creates a Raft node. Log indexes are 0-based (educational; production Raft is often 1-based).
func New(id string, peers []string, transport Transport, applyCh chan interface{}) *Node {
	return &Node{
		id:          id,
		peers:       peers,
		state:       Follower,
		log:         make([]LogEntry, 0),
		commitIndex: -1,
		lastApplied: -1,
		transport:   transport,
		stopCh:      make(chan struct{}),
		applyCh:     applyCh,
		nextIndex:   make(map[string]int),
		matchIndex:  make(map[string]int),
	}
}

// Start begins the educational state machine loop.
func (n *Node) Start() {
	go n.run()
}

// Stop signals the node to exit its run loop.
func (n *Node) Stop() {
	select {
	case <-n.stopCh:
	default:
		close(n.stopCh)
	}
}

// State returns the current node state.
func (n *Node) State() State {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state
}

// Term returns the current term.
func (n *Node) Term() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.currentTerm
}

// CommitIndex returns the highest committed log index (-1 if empty).
func (n *Node) CommitIndex() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.commitIndex
}

// Log returns a copy of the in-memory log.
func (n *Node) Log() []LogEntry {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]LogEntry, len(n.log))
	copy(out, n.log)
	return out
}

// Propose appends a command to the leader log and attempts replication.
// Returns ErrNotLeader if this node is not currently the leader.
func (n *Node) Propose(command interface{}) error {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return ErrNotLeader
	}
	entry := LogEntry{Term: n.currentTerm, Command: command}
	n.log = append(n.log, entry)
	n.mu.Unlock()

	n.replicateToFollowers()
	return nil
}

// HandleAppendEntries applies a leader AppendEntries RPC on a follower/candidate.
// Returns (term, success) matching the Transport.AppendEntries contract.
func (n *Node) HandleAppendEntries(term int, leaderID string, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int) (int, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term < n.currentTerm {
		return n.currentTerm, false
	}
	if term > n.currentTerm {
		n.currentTerm = term
		n.votedFor = ""
	}
	n.state = Follower

	// Prev log consistency check (educational 0-based indexing).
	if prevLogIndex >= 0 {
		if prevLogIndex >= len(n.log) {
			return n.currentTerm, false
		}
		if n.log[prevLogIndex].Term != prevLogTerm {
			return n.currentTerm, false
		}
	} else if prevLogIndex < -1 {
		return n.currentTerm, false
	}

	// Append new entries, truncating conflicts.
	insertAt := prevLogIndex + 1
	for i, e := range entries {
		idx := insertAt + i
		if idx < len(n.log) {
			if n.log[idx].Term != e.Term {
				n.log = n.log[:idx]
				n.log = append(n.log, entries[i:]...)
				break
			}
			continue
		}
		n.log = append(n.log, entries[i:]...)
		break
	}

	if leaderCommit > n.commitIndex {
		lastNew := len(n.log) - 1
		if leaderCommit < lastNew {
			n.commitIndex = leaderCommit
		} else {
			n.commitIndex = lastNew
		}
		n.applyCommittedLocked()
	}
	_ = leaderID
	return n.currentTerm, true
}

// HandleRequestVote applies a RequestVote RPC.
func (n *Node) HandleRequestVote(term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term < n.currentTerm {
		return n.currentTerm, false
	}
	if term > n.currentTerm {
		n.currentTerm = term
		n.votedFor = ""
		n.state = Follower
	}

	lastIdx := len(n.log) - 1
	lastTerm := 0
	if lastIdx >= 0 {
		lastTerm = n.log[lastIdx].Term
	}
	upToDate := lastLogTerm > lastTerm || (lastLogTerm == lastTerm && lastLogIndex >= lastIdx)
	if (n.votedFor == "" || n.votedFor == candidateID) && upToDate {
		n.votedFor = candidateID
		return n.currentTerm, true
	}
	return n.currentTerm, false
}

func (n *Node) applyCommittedLocked() {
	for n.lastApplied < n.commitIndex {
		n.lastApplied++
		if n.applyCh != nil {
			cmd := n.log[n.lastApplied].Command
			select {
			case n.applyCh <- cmd:
			default:
			}
		}
	}
}

func (n *Node) run() {
	for {
		select {
		case <-n.stopCh:
			return
		default:
		}
		switch n.State() {
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
	defer timer.Stop()

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
	lastLogIndex := len(n.log) - 1
	lastLogTerm := 0
	if lastLogIndex >= 0 {
		lastLogTerm = n.log[lastLogIndex].Term
	}
	peers := append([]string(nil), n.peers...)
	n.mu.Unlock()

	notifyCh := make(chan struct{}, 1)
	votes := 1
	var votesMu sync.Mutex

	for _, peer := range peers {
		go func(p string) {
			t, granted := n.transport.RequestVote(p, term, n.id, lastLogIndex, lastLogTerm)
			if granted {
				votesMu.Lock()
				votes++
				majority := votes > (len(peers)+1)/2
				votesMu.Unlock()
				if majority {
					n.mu.Lock()
					if n.state == Candidate && n.currentTerm == term {
						n.state = Leader
						for _, peerID := range n.peers {
							n.nextIndex[peerID] = len(n.log)
							n.matchIndex[peerID] = -1
						}
						select {
						case notifyCh <- struct{}{}:
						default:
						}
					}
					n.mu.Unlock()
				}
			} else if t > term {
				n.mu.Lock()
				n.state = Follower
				n.currentTerm = t
				n.votedFor = ""
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
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			if n.State() != Leader {
				return
			}
			n.replicateToFollowers()
		}
	}
}

// replicateToFollowers sends AppendEntries (heartbeats or log batches) to peers.
func (n *Node) replicateToFollowers() {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return
	}
	term := n.currentTerm
	leaderID := n.id
	commit := n.commitIndex
	peers := append([]string(nil), n.peers...)
	next := make(map[string]int, len(n.nextIndex))
	for k, v := range n.nextIndex {
		next[k] = v
	}
	logCopy := make([]LogEntry, len(n.log))
	copy(logCopy, n.log)
	n.mu.Unlock()

	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			ni := next[p]
			prevLogIndex := ni - 1
			prevLogTerm := 0
			if prevLogIndex >= 0 && prevLogIndex < len(logCopy) {
				prevLogTerm = logCopy[prevLogIndex].Term
			}
			var entries []LogEntry
			if ni < len(logCopy) {
				entries = append([]LogEntry(nil), logCopy[ni:]...)
			}
			t, ok := n.transport.AppendEntries(p, term, leaderID, prevLogIndex, prevLogTerm, entries, commit)
			n.mu.Lock()
			defer n.mu.Unlock()
			if t > n.currentTerm {
				n.state = Follower
				n.currentTerm = t
				n.votedFor = ""
				return
			}
			if n.state != Leader || n.currentTerm != term {
				return
			}
			if ok {
				n.nextIndex[p] = len(n.log)
				n.matchIndex[p] = len(n.log) - 1
				n.advanceCommitLocked()
			} else if n.nextIndex[p] > 0 {
				n.nextIndex[p]--
			}
		}(peer)
	}
	wg.Wait()
}

func (n *Node) advanceCommitLocked() {
	// Find highest N such that a majority have matchIndex >= N and log[N].Term == currentTerm.
	for N := len(n.log) - 1; N > n.commitIndex; N-- {
		if N < 0 {
			break
		}
		if n.log[N].Term != n.currentTerm {
			continue
		}
		count := 1 // leader
		for _, p := range n.peers {
			if n.matchIndex[p] >= N {
				count++
			}
		}
		if count > (len(n.peers)+1)/2 {
			n.commitIndex = N
			n.applyCommittedLocked()
			return
		}
	}
}

func randomTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
