/*
Package ipfs provides an IPFS HTTP API client scaffold.

Prefer pkg/web3.Store and adapters/memory for new code; this package is not yet
adapted behind the root Store interface.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/web3/storage/ipfs"

	client, err := ipfs.New(ipfs.Config{APIURL: "http://localhost:5001"})
	cid, err := client.Add(ctx, data)
*/
package ipfs
