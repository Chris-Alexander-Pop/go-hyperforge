/*
Package paxos is an educational sketch of single-decree Paxos for learning and
API exploration only.

Proposer/Acceptor helpers demonstrate prepare/accept phases against an
abstract Transport. This is not Multi-Paxos, has no durable storage, and is
not a production consensus implementation. Prefer etcd/raft, HashiCorp raft,
or a managed consensus service for real systems.
*/
package paxos
