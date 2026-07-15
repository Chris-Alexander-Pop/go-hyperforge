// Package lvm provides a local/LVM-shaped VolumeController for tests and dev.
//
// Volumes are sparse files under a root directory with JSON metadata. This does
// not shell out to real LVM (lvcreate/lvremove); it is intentionally local-enough
// for unit and integration tests of the controller interface.
package lvm
