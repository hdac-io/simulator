package network

import (
	"net"
	"strconv"
	"strings"
	"sync/atomic"
)

var uniquePort int32 = 9000

type tcpAddress struct {
	port     int
	listener net.Listener
}

// NewTCPAddress construct Address struct
func NewTCPAddress() Address {
	port := int(uniquePort)
	address := tcpAddress{
		port: port,
	}
	// FIXME: error handling
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(err)
	}
	address.listener = listener
	atomic.AddInt32(&uniquePort, 1)

	return address
}

// Listen waits connection request
func (a tcpAddress) Listen() Network {
	// FIXME: error handling
	conn, err := a.listener.Accept()
	if err != nil {
		panic(err)
	}

	remote := strings.Split(conn.RemoteAddr().String(), ":")
	remoteAddress := tcpAddress{}
	remoteAddress.port, _ = strconv.Atoi(remote[1])

	return newTCPNetwork(remoteAddress, conn)
}

// Connect construct connection to destination
func (a tcpAddress) Connect(destination Address) Network {
	conn, err := net.Dial("tcp", ":"+strconv.Itoa(destination.(tcpAddress).port))
	if err != nil {
		panic(err)
	}

	return newTCPNetwork(destination, conn)
}
