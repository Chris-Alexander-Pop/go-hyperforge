package raft

import (
	"sync"
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
	peers := []string{"peer1", "peer2"}

	transport := &mockTransport{
		requestVoteFunc: func(peer string, term int, candidateID string, lastLogIndex int, lastLogTerm int) (int, bool) {
			return term, true
		},
	}

	n := New("node1", peers, transport, nil)
	n.state = Candidate

	start := time.Now()
	n.runCandidate()
	duration := time.Since(start)

	if n.State() != Leader {
		t.Errorf("Expected state to be Leader, got %v", n.State())
	}
	if duration > 50*time.Millisecond {
		t.Errorf("Election took too long: %v (expected < 50ms)", duration)
	}
}

func TestProposeNotLeader(t *testing.T) {
	n := New("n1", nil, &mockTransport{}, nil)
	if err := n.Propose("x"); err != ErrNotLeader {
		t.Fatalf("expected ErrNotLeader, got %v", err)
	}
}

func TestProposeAndReplicate(t *testing.T) {
	follower := New("f1", nil, &mockTransport{}, make(chan interface{}, 8))
	var mu sync.Mutex
	var sent []LogEntry

	leaderTransport := &mockTransport{
		appendEntriesFunc: func(peer string, term int, leaderID string, prevLogIndex int, prevLogTerm int, entries []LogEntry, leaderCommit int) (int, bool) {
			mu.Lock()
			sent = append(sent, entries...)
			mu.Unlock()
			return follower.HandleAppendEntries(term, leaderID, prevLogIndex, prevLogTerm, entries, leaderCommit)
		},
	}

	leader := New("l1", []string{"f1"}, leaderTransport, make(chan interface{}, 8))
	leader.mu.Lock()
	leader.state = Leader
	leader.currentTerm = 1
	leader.nextIndex["f1"] = 0
	leader.matchIndex["f1"] = -1
	leader.mu.Unlock()

	if err := leader.Propose("cmd-a"); err != nil {
		t.Fatal(err)
	}
	if err := leader.Propose("cmd-b"); err != nil {
		t.Fatal(err)
	}

	flog := follower.Log()
	if len(flog) != 2 {
		t.Fatalf("follower log len=%d want 2", len(flog))
	}
	if flog[0].Command != "cmd-a" || flog[1].Command != "cmd-b" {
		t.Fatalf("follower log=%v", flog)
	}
	if leader.CommitIndex() < 1 {
		t.Fatalf("leader commitIndex=%d want >=1", leader.CommitIndex())
	}
	if follower.CommitIndex() < 0 {
		t.Fatalf("follower should have advanced commit")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sent) == 0 {
		t.Fatal("expected AppendEntries with entries")
	}
}

func TestHandleAppendEntriesRejectsStaleTerm(t *testing.T) {
	n := New("n1", nil, &mockTransport{}, nil)
	n.mu.Lock()
	n.currentTerm = 5
	n.mu.Unlock()
	term, ok := n.HandleAppendEntries(3, "L", -1, 0, []LogEntry{{Term: 3, Command: "x"}}, 0)
	if ok || term != 5 {
		t.Fatalf("got term=%d ok=%v", term, ok)
	}
}

func TestHandleAppendEntriesPrevLogMismatch(t *testing.T) {
	n := New("n1", nil, &mockTransport{}, nil)
	n.mu.Lock()
	n.currentTerm = 1
	n.log = []LogEntry{{Term: 1, Command: "a"}}
	n.mu.Unlock()
	_, ok := n.HandleAppendEntries(1, "L", 0, 99, []LogEntry{{Term: 1, Command: "b"}}, 0)
	if ok {
		t.Fatal("expected prev log term mismatch rejection")
	}
}
