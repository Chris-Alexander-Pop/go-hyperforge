// Package scheduler provides the logic for placing workloads onto hosts.
//
// Shipping: memory adapter with real strategies — "binpack", "spread", "random".
// Production cluster schedulers (Kubernetes scheduler plugins, etc.) are not wired.
package scheduler
