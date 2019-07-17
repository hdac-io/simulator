package tcp

import (
	"net"

	mynet "github.com/hdac-io/simulator/net"
)

// Network represents TCP network
type Network struct {
	address  *net.TCPAddr
	listener net.Listener
}

// New construct Network struct
func New(address mynet.Address) Network {
	addr, err := net.ResolveTCPAddr("tcp", address.(string))
	if err != nil {
		panic(err)
	}
	network := Network{
		address: addr,
	}
	// FIXME: error handling
	listener, err := net.Listen("tcp", addr.String())
	if err != nil {
		panic(err)
	}
	network.listener = listener

	return network
}

// Accept waits connection request
func (n Network) Accept() mynet.Connection {
	// FIXME: error handling
	conn, err := n.listener.Accept()
	if err != nil {
		panic(err)
	}

	remoteAddress := Network{
		address: conn.RemoteAddr().(*net.TCPAddr),
	}

	return newConnection(remoteAddress, conn)
}

// Connect construct connection to destination
func Connect(destination mynet.Address) mynet.Connection {
	conn, err := net.Dial("tcp", destination.(string))
	// FIXME: error handling
	if err != nil {
		panic(err)
	}

	return newConnection(destination, conn)
}
