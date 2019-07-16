package tcp

import (
	"net"

	mynet "github.com/hdac-io/simulator/net"
)

type network struct {
	address  *net.TCPAddr
	listener net.Listener
}

// New construct Network struct
func New(address mynet.Address) mynet.Network {
	addr, err := net.ResolveTCPAddr("tcp", address.(string))
	if err != nil {
		panic(err)
	}
	network := network{
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

// Listen waits connection request
func (n network) Accept() mynet.Connection {
	// FIXME: error handling
	conn, err := n.listener.Accept()
	if err != nil {
		panic(err)
	}

	remoteAddress := network{
		address: conn.RemoteAddr().(*net.TCPAddr),
	}

	return newConnection(remoteAddress, conn)
}

// Connect construct connection to destination
func (n network) Connect(destination mynet.Address) mynet.Connection {
	conn, err := net.Dial("tcp", destination.(string))
	// FIXME: error handling
	if err != nil {
		panic(err)
	}

	return newConnection(destination, conn)
}
