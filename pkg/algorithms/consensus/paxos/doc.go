/*
Package paxos is an educational sketch of single-decree and Multi-Paxos for
learning and API exploration.

Proposer/Acceptor/Learner helpers demonstrate prepare/accept/learn against an
abstract Transport. MultiPaxos sequences decrees into slots. This has no durable
storage, leader election, or reconfiguration, and is not a production consensus
implementation. Prefer etcd/raft, HashiCorp raft, or a managed consensus service
for real systems.
*/
package paxos
