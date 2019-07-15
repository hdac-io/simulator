package network

import (
	"encoding/gob"
	"io"
	"net"

	"github.com/hdac-io/simulator/signature"

	"github.com/hdac-io/simulator/block"
)

// VirtualNetwork represents virtual public network
type tcpNetwork struct {
	address    Address
	connection net.Conn
	encoder    *gob.Encoder
	decoder    *gob.Decoder
}

func newTCPNetwork(address Address, conn net.Conn) *tcpNetwork {
	if conn == nil {
		panic("Connection is nil !!")
	}

	gob.Register(block.Block{})
	gob.Register(signature.Signature{})

	return &tcpNetwork{
		address:    address,
		connection: conn,
		encoder:    gob.NewEncoder(conn),
		decoder:    gob.NewDecoder(conn),
	}
}

type packet struct {
	Load load
}

// Write load to virtual network
func (n *tcpNetwork) Write(l load) {
	go func() {
		err := n.encoder.Encode(packet{Load: l})
		if err != nil {
			panic(err)
		}
	}()
}

// Read load from virtual network
func (n *tcpNetwork) Read() load {
	p := packet{}
	err := n.decoder.Decode(&p)
	if err != nil && err != io.EOF {
		panic(err)
	}

	return p.Load
}

// GetAddress retrieves network address
func (n *tcpNetwork) GetAddress() Address {
	return n.address
}
