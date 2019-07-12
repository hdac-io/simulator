package node

import (
	"sync/atomic"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/signature"
)

// channel represents inbound and outbound channel
type channel struct {
	address network.Address
	peers   map[int]*peer

	// for inbound
	block     chan block.Block
	signature chan signature.Signature
}

type peer struct {
	network network.Network
}

func newPeer(network network.Network) *peer {
	return &peer{
		network: network,
	}
}

var unique int32

// newChannel construct channel
func newChannel() *channel {
	c := channel{
		address:   network.NewAddress(int(unique)),
		peers:     make(map[int]*peer),
		block:     make(chan block.Block, 1024),
		signature: make(chan signature.Signature, 1024),
	}
	atomic.AddInt32(&unique, 1)

	// Start connection listener
	c.startConnectionListner()

	return &c
}

func (c *channel) addPeer(destination network.Address) {
	_, exist := c.peers[destination.Unique]
	if exist {
		return
	}

	dest := c.address.Connect(destination)
	peer := newPeer(dest)
	c.setPeer(peer)
}

func (c *channel) sendSignature(sign signature.Signature) {
	for _, peer := range c.peers {
		peer.network.Write(sign)
	}
}

func (c *channel) sendBlock(b block.Block) {
	for _, peer := range c.peers {
		peer.network.Write(b)
	}
}

func (c *channel) readSignature() signature.Signature {
	return <-c.signature
}

func (c *channel) readBlock() block.Block {
	return <-c.block
}

func (c *channel) startConnectionListner() {
	go func() {
		for {
			dest := c.address.Listen()
			peer := newPeer(dest)
			c.setPeer(peer)
		}
	}()
}

func (c *channel) setPeer(p *peer) {
	_, exist := c.peers[p.network.Unique]
	if !exist {
		c.peers[p.network.Unique] = p

		// Start reader
		go func() {
			for {
				load := p.network.Read()
				switch v := load.(type) {
				case block.Block:
					c.block <- v
				case signature.Signature:
					c.signature <- v
				}
			}
		}()
	}
}
