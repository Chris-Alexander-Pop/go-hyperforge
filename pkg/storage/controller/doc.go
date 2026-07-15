// Package controller provides the abstraction for managing storage volumes.
//
// Shipping backends:
//   - adapters/memory — in-process VolumeController for unit tests
//   - adapters/lvm — local sparse-file + JSON meta controller (LVM-shaped, no real lvcreate)
//
// Planned: Ceph RBD, cloud CSI drivers.
package controller
