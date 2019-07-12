package network

// Address represents network address
type Address interface {
	Connect(destination Address) Network
	Listen() Network
}

type load = interface{}

// Network represents virtual public network
type Network interface {
	Read() load
	Write(l load)
	GetAddress() Address
}

func new(address Address) Network {
	return newVirtualNetwork(address)
}

func newLoopback(address Address) Network {
	return newVirtualLoopbackNetwork(address)
}
