package raft

import (
	"testing"
	"time"
)

type mockTransport struct {
	requestVoteFunc   func(peer string, term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool)
	appendEntriesFunc func(peer string, term int, leaderID string, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int) (int, bool)
}

func (m *mockTransport) RequestVote(peer string, term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool) {
	if m.requestVoteFunc != nil {
		return m.requestVoteFunc(peer, term, candidateID, lastLogIndex, lastLogTerm)
	}
	return term, false
}

func (m *mockTransport) AppendEntries(peer string, term int, leaderID string, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int) (int, bool) {
	if m.appendEntriesFunc != nil {
		return m.appendEntriesFunc(peer, term, leaderID, prevLogIndex, prevLogTerm, entries, leaderCommit)
	}
	return term, false
}

func TestCandidateElectionSpeed(t *testing.T) {
	// 3 nodes, need 2 votes (1 self + 1 peer)
	peers := []string{"peer1", "peer2"}

	transport := &mockTransport{
		requestVoteFunc: func(peer string, term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool) {
			// Grant vote immediately
			return term, true
		},
	}

	n := New("node1", peers, transport, nil)
	n.state = Candidate

	start := time.Now()

	// We run runCandidate.
	// We want to assert that it returns quickly when votes are granted.
	n.runCandidate()

	duration := time.Since(start)

	// Check if we became leader
	n.mu.Lock()
	state := n.state
	n.mu.Unlock()

	if state != Leader {
		t.Errorf("Expected state to be Leader, got %v", state)
	}

	t.Logf("Election took %v", duration)

	// We expect this to be very fast (<< 50ms).
	// Using 50ms as a safe upper bound.
	if duration > 50*time.Millisecond {
		t.Errorf("Election took too long: %v (expected < 50ms)", duration)
	}
}
