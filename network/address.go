package network

// Address represents network address
type Address struct {
	Unique  int
	network chan Network
}

// NewAddress construct Address struct
func NewAddress(unique int) Address {
	return Address{
		Unique:  unique,
		network: make(chan Network),
	}
}

// Listen waits connection request
func (a Address) Listen() Network {
	return <-a.network
}

// Connect construct connection to destination
func (a Address) Connect(destination Address) Network {
	var network Network
	if a.Unique == destination.Unique {
		network = NewLoopbackNetwork(destination)
	} else {
		network = NewNetwork(destination)

		// swap inbound and outbound
		destNetwork := Network{
			Unique:   a.Unique,
			inbound:  network.outbound,
			outbound: network.inbound,
			getDelay: network.getDelay,
		}
		destination.network <- destNetwork
	}

	return network
}
