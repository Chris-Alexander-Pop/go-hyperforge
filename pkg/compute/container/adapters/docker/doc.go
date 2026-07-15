// Package docker provides a Docker Engine API adapter for container.ContainerRuntime.
//
// Talks to the Docker daemon via the official client SDK (unix socket or DOCKER_HOST).
// Inject ContainerAPI for unit tests without a running daemon.
package docker
