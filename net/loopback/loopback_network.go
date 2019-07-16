package loopback

import (
	"github.com/hdac-io/simulator/net"
)

// Network represents loopback network
type network struct {
}

// New construct Network struct
func New() net.Network {
	return network{}
}

// Accept waits connection request
func (n network) Accept() net.Connection {
	// Dummy
	return nil
}

// Connect construct connection to destination
func (n network) Connect(destination net.Address) net.Connection {
	// FIXME: should confirm destination
	return newConnection()
}
