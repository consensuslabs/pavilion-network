package main

import (
	"context"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// PeerDiscovery implements peer discovery using mDNS
type PeerDiscovery struct {
	ctx        context.Context
	host       host.Host
	rendezvous string
	peerChan   chan peer.AddrInfo
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// HandlePeerFound implements the Notifee interface
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

// NewPeerDiscovery creates a new Discovery service
func NewPeerDiscovery(ctx context.Context, h host.Host, rendezvous string) (*PeerDiscovery, error) {
	// Create a new PeerChan for newly discovered peers
	peerChan := make(chan peer.AddrInfo)

	// Create a new Discovery instance
	d := &PeerDiscovery{
		ctx:        ctx,
		host:       h,
		rendezvous: rendezvous,
		peerChan:   peerChan,
	}

	// Initialize the mDNS service
	notifee := &discoveryNotifee{PeerChan: peerChan}
	mdnsService := mdns.NewMdnsService(h, rendezvous, notifee)
	if err := mdnsService.Start(); err != nil {
		return nil, err
	}

	return d, nil
}

// Advertise starts advertising this node's presence
func (d *PeerDiscovery) Advertise() {
	// mDNS discovery automatically advertises peer presence
	log.Printf("Advertising peer %s", d.host.ID().String())
}

// FindPeers continuously looks for peers
func (d *PeerDiscovery) FindPeers() {
	go func() {
		for {
			select {
			case peer := <-d.peerChan:
				// Skip connecting to self
				if peer.ID == d.host.ID() {
					continue
				}

				log.Printf("Found peer: %s", peer.ID.String())

				// Try to connect to the discovered peer
				if err := d.host.Connect(d.ctx, peer); err != nil {
					log.Printf("Failed to connect to peer %s: %s", peer.ID.String(), err)
					continue
				}

				log.Printf("Connected to peer: %s", peer.ID.String())

			case <-d.ctx.Done():
				return
			}
		}
	}()
}

// GetConnectedPeers returns a list of currently connected peers
func (d *PeerDiscovery) GetConnectedPeers() []peer.ID {
	return d.host.Network().Peers()
}

// WaitForPeers blocks until the node has connected to at least minPeers
func (d *PeerDiscovery) WaitForPeers(minPeers int) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return d.ctx.Err()
		case <-ticker.C:
			if len(d.GetConnectedPeers()) >= minPeers {
				return nil
			}
		}
	}
}
