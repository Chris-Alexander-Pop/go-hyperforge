/*
Package ethereum provides a go-ethereum ethclient wrapper for EVM chains.

This is an SDK-coupled scaffold. Prefer pkg/web3.Client and adapters/memory for
new code; this package is not yet adapted behind the root Client interface.

Usage:

	import "github.com/chris-alexander-pop/system-design-library/pkg/web3/blockchain/ethereum"

	client, err := ethereum.New(ethereum.Config{RPCURL: "https://mainnet.infura.io/v3/..."})
	balance, err := client.GetBalance(ctx, "0x...")
*/
package ethereum
