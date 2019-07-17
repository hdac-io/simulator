package node

import (
	"sync"

	"github.com/hdac-io/simulator/block"
	"github.com/hdac-io/simulator/net"
	"github.com/hdac-io/simulator/net/loopback"
	"github.com/hdac-io/simulator/net/tcp"
	"github.com/hdac-io/simulator/signature"
	"github.com/hdac-io/simulator/types"
)

// channel represents inbound and outbound channel
type channel struct {
	sync.Mutex
	id    types.ID
	tcp   tcp.Network
	peers map[net.Address]*peer

	// for inbound
	block     chan block.Block
	signature chan signature.Signature
}

type peer struct {
	connection net.Connection
}

func newPeer(network net.Connection) *peer {
	return &peer{
		connection: network,
	}
}

// newChannel construct channel
func newChannel(myaddr address) *channel {
	c := channel{
		id:        myaddr.ID,
		tcp:       tcp.New(myaddr.Address),
		peers:     make(map[net.Address]*peer),
		block:     make(chan block.Block, 1024),
		signature: make(chan signature.Signature, 1024),
	}

	// Start connection listener
	c.startConnectionListner()

	return &c
}

func (c *channel) addKnownPeers(addressbook Addressbook) {
	// Add loopback
	connection := loopback.Connect("loopback")
	peer := newPeer(connection)
	c.setPeer(peer)

	// FIXME
	// Node has higher ID connect to nodes have lower ID
	for id, address := range addressbook {
		if id != address.ID {
			panic("Invalid address !")
		}
		if address.ID < c.id {
			c.addPeer(address.Address)
		}
	}
}

func (c *channel) addPeer(destination net.Address) {
	// FIXME: very naive locking mechanism
	c.Lock()
	_, exist := c.peers[destination]
	if exist {
		c.Unlock()
		return
	}
	c.Unlock()

	dest := tcp.Connect(destination)
	peer := newPeer(dest)
	c.setPeer(peer)
}

func (c *channel) sendSignature(sign signature.Signature) {
	for _, peer := range c.peers {
		peer.connection.Write(sign)
	}
}

func (c *channel) sendBlock(b block.Block) {
	for _, peer := range c.peers {
		peer.connection.Write(b)
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
			dest := c.tcp.Accept()
			peer := newPeer(dest)
			c.setPeer(peer)
		}
	}()
}

func (c *channel) setPeer(p *peer) {
	address := p.connection.GetAddress()
	// FIXME: very naive locking mechanism
	c.Lock()
	defer c.Unlock()
	_, exist := c.peers[address]
	if !exist {
		c.peers[address] = p

		// Start reader
		go func() {
			for {
				load := p.connection.Read()
				switch v := load.(type) {
				case block.Block:
					c.block <- v
				case signature.Signature:
					c.signature <- v
				}
			}
		}()
	} else {
		panic("Cannot enter here !")
	}
}
