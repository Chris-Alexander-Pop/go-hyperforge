// Package ebs provides a file-like AWS EBS stub implementing block.VolumeStore.
//
// Volume/snapshot metadata is persisted as JSON under a root directory with
// vol-/snap- style IDs. This is not a real EC2/EBS API client — use it for
// local/dev and unit tests. Inject a custom Root via Config.
package ebs
