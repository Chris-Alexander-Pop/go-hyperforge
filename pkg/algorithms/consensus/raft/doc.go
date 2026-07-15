/*
Package raft is an educational, incomplete Raft consensus sketch.

What works (in-memory, mock Transport):
  - Follower → Candidate → Leader election via RequestVote
  - Leader Propose appends a LogEntry and replicates via AppendEntries
  - Followers apply HandleAppendEntries (prev-log check, truncate/append, commit advance)
  - Leaders advance commitIndex when a majority matches

What does NOT work / is intentionally incomplete:
  - No durable storage or snapshots
  - No real network / RPC framing
  - Election timer is not reset by AppendEntries in the run loop
  - No membership changes, linearizability guarantees, or production safety audit

Prefer etcd/raft or HashiCorp raft for production systems. This package exists
for learning and unit tests only.
*/
package raft
