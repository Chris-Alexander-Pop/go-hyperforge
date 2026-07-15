// Package local provides a real local filesystem FileStore adapter.
//
// Point MountPoint at a local directory or an already-mounted NFS/EFS path.
// This package does not invoke mount(8); it only reads/writes under the root.
package local
