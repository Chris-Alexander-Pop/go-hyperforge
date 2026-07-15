// Package redfish provides a Redfish BMC adapter for provisioning.Provisioner.
//
// PowerCycle and GetHostStatus talk to a Redfish-compatible HTTP endpoint
// (mockable via httptest). ProvisionHost/DeprovisionHost return Unimplemented
// (OS imaging is out of band).
package redfish
