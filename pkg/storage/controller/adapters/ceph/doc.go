// Package ceph provides a Ceph RBD-shaped VolumeController.
//
// Honesty: this adapter does not link against librados/librbd. It speaks an
// injectable RBDClient interface (HTTP/rados-shaped) so unit tests use an
// in-process MemoryRBDClient. Wire a real HTTP gateway or thin rados wrapper
// behind RBDClient for production.
package ceph
