package loopback

import "github.com/hdac-io/simulator/net"

type connection struct {
	address  string
	loopback chan net.Load
}

func newConnection() connection {
	return connection{
		// FIXME: type
		address:  "loopback",
		loopback: make(chan net.Load, 16),
	}
}

// Write load to loopback network
func (c connection) Write(l net.Load) {
	c.loopback <- l
}

// Read load from loopback network
func (c connection) Read() net.Load {
	return <-c.loopback
}

// GetAddress retrieves network address
func (c connection) GetAddress() net.Address {
	return c.address
}
