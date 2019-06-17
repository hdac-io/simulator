package fridayconsensus

import (
	"simulator/network"
)

// channel represents inbound and outbound channel
type channel struct {
	inbound  peer
	outbound []peer
}

type peer struct {
	block     chan block
	signature chan signature
	network   *network.Network
}

func newPeer(network *network.Network) peer {
	return peer{
		block:     make(chan block, 1024),
		signature: make(chan signature, 1024),
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
			case block:
				c.inbound.block <- v
			case signature:
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

func (c *channel) sendSignature(sig signature) {
	for _, out := range c.outbound {
		out.signature <- sig
	}
}

func (c *channel) sendBlock(b block) {
	for _, out := range c.outbound {
		out.block <- b
	}
}

func (c *channel) readSignature() signature {
	return <-c.inbound.signature
}

func (c *channel) readBlock() block {
	return <-c.inbound.block
}
