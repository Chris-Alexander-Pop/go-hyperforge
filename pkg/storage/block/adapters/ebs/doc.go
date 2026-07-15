// Package ebs provides AWS EBS-shaped block.VolumeStore adapters.
//
// Shipped:
//   - File stub (New / NewWithBlockConfig): local JSON volumes/snapshots with vol-/snap- IDs
//   - SDK store (NewSDK / NewSDKFromAPI): real EC2 CreateVolume/AttachVolume/… via aws-sdk-go-v2
//
// Remaining gaps (honest): Azure Managed Disks / GCP PD / Ceph / multi-attach /
// waiters for volume state transitions are not wrapped.
package ebs
