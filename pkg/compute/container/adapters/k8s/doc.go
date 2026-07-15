// Package k8s provides a Kubernetes adapter for container.ContainerRuntime.
//
// Container.ID is the pod name so Create returns an ID usable with Get and
// other lifecycle methods. Exec and Stats are stubs (no metrics-server / SPDY).
package k8s
