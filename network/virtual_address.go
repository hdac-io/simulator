package network

import "sync/atomic"

var unique int32

type virtualAddress struct {
	unique  int
	network chan Network
}

// NewAddress construct Address struct
func NewAddress() Address {
	address := virtualAddress{
		unique:  int(unique),
		network: make(chan Network),
	}
	atomic.AddInt32(&unique, 1)

	return address
}

// Listen waits connection request
func (a virtualAddress) Listen() Network {
	return <-a.network
}

// Connect construct connection to destination
func (a virtualAddress) Connect(destination Address) Network {
	var network *VirtualNetwork
	dest := destination.(virtualAddress)
	if a.unique == dest.unique {
		network = newLoopback(destination).(*VirtualNetwork)
	} else {
		network = new(destination).(*VirtualNetwork)

		// swap inbound and outbound
		destNetwork := VirtualNetwork{
			address:  a,
			inbound:  network.outbound,
			outbound: network.inbound,
			getDelay: network.getDelay,
		}
		dest.network <- &destNetwork
	}

	return network
}
