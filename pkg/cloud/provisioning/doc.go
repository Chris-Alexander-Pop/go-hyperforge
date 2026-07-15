// Package provisioning provides interfaces for managing the lifecycle of physical hardware.
//
// Shipping:
//   - memory adapter
//   - Redfish BMC PowerCycle / GetHostStatus (adapters/redfish)
//   - IPMI-over-LAN HTTP gateway (adapters/ipmi)
//
// PXE/iPXE imaging remains reserved.
package provisioning
