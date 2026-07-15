// Package memory provides an in-memory implementation of loadbalancer.LoadBalancerManager
// and loadbalancer.TargetSelector.
//
// SelectTarget delegates to pkg/algorithms/loadbalancing strategies based on the
// pool Algorithm (round-robin, least-connections, weighted round-robin, random).
package memory
