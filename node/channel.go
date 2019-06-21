package node

import (
	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/signature"
)

// channel represents inbound and outbound channel
type channel struct {
	inbound  peer
	outbound []peer
}

type peer struct {
	block     chan block.Block
	signature chan signature.Signature
	network   *network.Network
}

func newPeer(network *network.Network) peer {
	return peer{
		block:     make(chan block.Block, 1024),
		signature: make(chan signature.Signature, 1024),
		network:   network,
	}
}

// newChannel construct channel
func newChannel() *channel {
	c := channel{
		inbound: newPeer(network.NewNetwork()),
	}

	return &c
}

// start starts Channel architecture
func (c *channel) start(peers []*network.Network) {
	// Start reader
	go func() {
		for {
			load := c.inbound.network.Read()
			switch v := load.(type) {
			case block.Block:
				c.inbound.block <- v
			case signature.Signature:
				c.inbound.signature <- v
			}
		}
	}()

	// Start writer
	for _, p := range peers {
		outbound := newPeer(p)
		go func(outbound peer) {
			for {
				select {
				case load := <-outbound.signature:
					outbound.network.Write(load)
				case load := <-outbound.block:
					outbound.network.Write(load)
				}
			}
		}(outbound)

		c.outbound = append(c.outbound, outbound)
	}
}

func (c *channel) sendSignature(sign signature.Signature) {
	for _, out := range c.outbound {
		out.signature <- sign
	}
}

func (c *channel) sendBlock(b block.Block) {
	for _, out := range c.outbound {
		out.block <- b
	}
}

func (c *channel) readSignature() signature.Signature {
	return <-c.inbound.signature
}

func (c *channel) readBlock() block.Block {
	return <-c.inbound.block
}
