package loopback

import (
	"github.com/hdac-io/simulator/net"
)

// Connect construct connection to destination
func Connect(destination net.Address) net.Connection {
	// FIXME: should confirm destination
	return newConnection()
}
