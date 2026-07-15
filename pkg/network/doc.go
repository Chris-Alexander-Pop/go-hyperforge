/*
Package network provides low-level TCP and UDP server implementations.

These are network primitives for building custom protocols.
For HTTP/REST APIs, use pkg/api instead.

Usage:

	import "github.com/chris-alexander-pop/go-hyperforge/pkg/network"

	// Create a TCP server
	server := network.NewTCPServer(network.Config{
	    Addr: ":9000",
	}, handler)

	// Start serving
	err := server.ListenAndServe(ctx)
*/
package network
