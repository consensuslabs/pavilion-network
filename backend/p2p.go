// p2p.go
package main

import (
	"context"
	"fmt"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
)

// startP2PHost creates and returns a new libp2p Host.
func startP2PHost() (host.Host, error) {
	// Create a new libp2p Host that listens on a random TCP port on all interfaces.
	ctx := context.Background()
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		return nil, err
	}
	return h, nil
}

// printHostInfo prints the host's peer ID and listening addresses.
func printHostInfo(h host.Host) {
	fmt.Println("p2p Host ID:", h.ID().Pretty())
	fmt.Println("Listening on addresses:")
	for _, addr := range h.Addrs() {
		// Construct the full multiaddress with the peer ID.
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), h.ID().Pretty())
		fmt.Println(" -", fullAddr)
	}
}
