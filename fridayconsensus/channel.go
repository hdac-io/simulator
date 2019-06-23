package fridayconsensus

import (
	"github.com/hdac-io/simulator/network"
	"github.com/hdac-io/simulator/types"
)

// channel represents inbound and outbound channel
type channel struct {
	inbound  peer
	outbound []peer
}

type peer struct {
	block     chan types.Block
	signature chan types.Signature
	vrf       chan vrfMessage
	network   *network.Network
}

func newPeer(network *network.Network) peer {
	return peer{
		block:     make(chan types.Block, 1024),
		signature: make(chan types.Signature, 1024),
		vrf:       make(chan vrfMessage, 1024),
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
			case types.Block:
				c.inbound.block <- v
			case types.Signature:
				c.inbound.signature <- v
			case vrfMessage:
				c.inbound.vrf <- v
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
				case load := <-outbound.vrf:
					outbound.network.Write(load)
				}
			}
		}(outbound)

		c.outbound = append(c.outbound, outbound)
	}
}

func (c *channel) sendSignature(sig types.Signature) {
	for _, out := range c.outbound {
		out.signature <- sig
	}
}

func (c *channel) sendBlock(b types.Block) {
	for _, out := range c.outbound {
		out.block <- b
	}
}

func (c *channel) sendVRFMessage(vrf vrfMessage) {
	for _, out := range c.outbound {
		out.vrf <- vrf
	}
}

func (c *channel) readSignature() types.Signature {
	return <-c.inbound.signature
}

func (c *channel) readBlock() types.Block {
	return <-c.inbound.block
}

func (c *channel) readVRFMessage() vrfMessage {
	return <-c.inbound.vrf
}
