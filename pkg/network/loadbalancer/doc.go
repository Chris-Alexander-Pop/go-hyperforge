// Package loadbalancer provides a unified interface for load balancer management.
//
// Supported backends:
//   - Memory: In-memory load balancer for testing and in-process selection
//   - ALB: AWS Application Load Balancer
//   - NLB: AWS Network Load Balancer
//   - GCLB: Google Cloud Load Balancing
//   - AzureLB: Azure Load Balancer
//
// In-process selection (memory adapter) reuses strategies from
// pkg/algorithms/loadbalancing for round-robin, least-connections,
// weighted round-robin, and random. Do not reimplement those algorithms here.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/network/loadbalancer/adapters/memory"
//
//	mgr := memory.New()
//	pool, _ := mgr.CreateTargetPool(ctx, loadbalancer.CreateTargetPoolOptions{
//		Name: "api", Protocol: loadbalancer.ProtocolHTTP, Port: 8080,
//		Algorithm: loadbalancer.AlgorithmRoundRobin,
//	})
//	_ = mgr.AddTarget(ctx, pool.ID, loadbalancer.Target{Address: "10.0.0.1", Port: 8080})
//	target, _ := mgr.SelectTarget(ctx, pool.ID)
package loadbalancer
