// Package controller provides the abstraction for managing storage volumes.
//
// Shipping backends:
//   - adapters/memory — in-process VolumeController for unit tests
//   - adapters/lvm — local sparse-file + JSON meta controller (LVM-shaped, no real lvcreate)
//   - adapters/ceph — Ceph RBD-shaped controller (injectable RBDClient; not librados)
//   - adapters/csi — CSI-shaped controller (injectable CSIControllerAPI; not real CSI gRPC)
package controller
