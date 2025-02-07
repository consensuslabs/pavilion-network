package main

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// P2P represents the P2P network node
type P2P struct {
	ctx       context.Context
	Host      host.Host
	PubSub    *pubsub.PubSub
	Discovery *PeerDiscovery
	Topics    map[string]*pubsub.Topic
}

// NewP2PNode creates a new P2P node
func NewP2PNode(ctx context.Context, port int, rendezvous string) (*P2P, error) {
	// Generate a new Ed25519 key pair for this host
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %v", err)
	}

	// Create a new PubSub service using GossipSub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub: %v", err)
	}

	// Initialize peer discovery
	discovery, err := NewPeerDiscovery(ctx, h, rendezvous)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discovery: %v", err)
	}

	p2pNode := &P2P{
		ctx:       ctx,
		Host:      h,
		PubSub:    ps,
		Discovery: discovery,
		Topics:    make(map[string]*pubsub.Topic),
	}

	// Start peer discovery
	discovery.Advertise()
	discovery.FindPeers()

	log.Printf("P2P Node started with ID: %s", h.ID().String())
	for _, addr := range h.Addrs() {
		log.Printf("Listening on: %s/p2p/%s", addr, h.ID().String())
	}

	return p2pNode, nil
}

// Subscribe joins a topic and returns its subscription and topic handle
func (p *P2P) Subscribe(topicName string) (*pubsub.Subscription, *pubsub.Topic, error) {
	topic, err := p.PubSub.Join(topicName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to join topic %s: %v", topicName, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to subscribe to topic %s: %v", topicName, err)
	}

	p.Topics[topicName] = topic
	return sub, topic, nil
}

// Publish sends data to a topic
func (p *P2P) Publish(topicName string, data []byte) error {
	topic, ok := p.Topics[topicName]
	if !ok {
		return fmt.Errorf("not subscribed to topic: %s", topicName)
	}

	return topic.Publish(p.ctx, data)
}

// ListPeers returns a list of peers subscribed to a topic
func (p *P2P) ListPeers(topicName string) []peer.ID {
	topic, ok := p.Topics[topicName]
	if !ok {
		return []peer.ID{}
	}
	return topic.ListPeers()
}

// WaitForPeers waits until we have connected to at least minPeers
func (p *P2P) WaitForPeers(minPeers int) error {
	return p.Discovery.WaitForPeers(minPeers)
}

// Close shuts down the P2P node
func (p *P2P) Close() error {
	return p.Host.Close()
}
