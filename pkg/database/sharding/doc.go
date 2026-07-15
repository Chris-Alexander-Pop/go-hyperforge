/*
Package sharding provides database sharding capabilities via consistent hashing.

ConsistentHash wraps pkg/algorithms/consistenthash/ring and implements Strategy.
Wire it into sql.NewSharded for multi-backend GetShard routing; single-instance
SQL adapters do not perform real key-based sharding on their own.
*/
package sharding
