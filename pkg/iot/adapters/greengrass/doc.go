// Package greengrass provides an AWS Greengrass V2 management client and an
// iot.Client adapter for edge messaging bridges.
//
// Management APIs (cores/components/deployments) remain on Client.
// Prefer NewAdapter for pkg/iot.Client messaging abstractions.
package greengrass
