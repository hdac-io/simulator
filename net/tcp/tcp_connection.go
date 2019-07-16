package tcp

import (
	"encoding/gob"
	"io"
	"net"

	mynet "github.com/hdac-io/simulator/net"
	"github.com/hdac-io/simulator/signature"

	"github.com/hdac-io/simulator/block"
)

type connection struct {
	address    mynet.Address
	connection net.Conn
	encoder    *gob.Encoder
	decoder    *gob.Decoder
}

func newConnection(address mynet.Address, conn net.Conn) connection {
	if conn == nil {
		panic("Connection is nil !!")
	}

	gob.Register(block.Block{})
	gob.Register(signature.Signature{})

	return connection{
		address:    address,
		connection: conn,
		encoder:    gob.NewEncoder(conn),
		decoder:    gob.NewDecoder(conn),
	}
}

type packet struct {
	Load mynet.Load
}

// Write load to TCP network
func (c connection) Write(l mynet.Load) {
	go func() {
		err := c.encoder.Encode(packet{Load: l})
		if err != nil {
			panic(err)
		}
	}()
}

// Read load from TCP network
func (c connection) Read() mynet.Load {
	p := packet{}
	err := c.decoder.Decode(&p)
	if err != nil && err != io.EOF {
		panic(err)
	}

	return p.Load
}

// GetAddress retrieves network address
func (c connection) GetAddress() mynet.Address {
	return c.address
}
