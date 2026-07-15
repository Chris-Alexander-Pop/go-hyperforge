// Package ebs provides a file-backed AWS EBS-shaped block.VolumeStore.
//
// Shipped: Create/Get/List/Delete/Resize volumes; Attach/Detach; Create/Get/Delete/List
// snapshots; CreateVolume from SnapshotID. IDs use vol-/snap- prefixes.
//
// Remaining gaps (honest): not a real EC2/EBS API client — no AWS SDK calls,
// encryption/KMS, multi-attach, or AZ capacity checks.
package ebs
