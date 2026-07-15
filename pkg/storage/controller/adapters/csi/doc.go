// Package csi provides a CSI-shaped VolumeController.
//
// Honesty: this adapter does not speak real CSI gRPC protobufs. It wraps a thin
// injectable CSIControllerAPI (gRPC-or-HTTP-like / in-process fake) that mirrors
// CreateVolume / DeleteVolume / ControllerPublish / ControllerUnpublish /
// ControllerExpandVolume. Map Attach→ControllerPublish and Detach→ControllerUnpublish.
package csi
