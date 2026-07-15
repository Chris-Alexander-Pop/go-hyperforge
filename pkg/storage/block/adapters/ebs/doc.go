// Package ebs provides AWS EBS-shaped block.VolumeStore adapters.
//
// Shipped:
//   - File stub (New / NewWithBlockConfig): local JSON volumes/snapshots with vol-/snap- IDs
//   - SDK store (NewSDK / NewSDKFromAPI): real EC2 CreateVolume/AttachVolume/… via aws-sdk-go-v2
//   - Volume/snapshot state waiters on SDKStore (WaitUntilVolumeAvailable / InUse / Deleted,
//     WaitUntilSnapshotCompleted); PollInterval defaults to 2s (injectable via SDKConfig)
//
// Remaining gaps (honest): Azure Managed Disks / GCP PD / Ceph / multi-attach are not wrapped here.
package ebs
