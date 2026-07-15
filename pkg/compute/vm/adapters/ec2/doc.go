// Package ec2 provides an AWS EC2 adapter for vm.VMManager.
//
// Usage:
//
//	import "github.com/chris-alexander-pop/go-hyperforge/pkg/compute/vm/adapters/ec2"
//
//	mgr, err := ec2.New(ec2.Config{Region: "us-east-1"})
//	inst, err := mgr.Create(ctx, vm.CreateOptions{ImageID: "ami-...", InstanceType: "t3.medium"})
//
// For tests, inject a mock via NewWithClient satisfying the EC2API interface.
package ec2
